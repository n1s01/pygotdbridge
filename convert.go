// Package telegotd конвертирует существующие Telethon-сессии в нативный
// формат gotd (github.com/gotd/td), позволяя работать аккаунтом через gotd без
// повторной авторизации.
//
// Поддерживаются два входных формата Telethon:
//
//   - SQLiteSession — дефолтный `.session` файл (SQLite);
//   - StringSession — компактная строка (декодируется штатной функцией gotd).
//
// Основная точка входа — StorageFromInput: на вход даётся сессия, на выход —
// готовый session.Storage для telegram.Options{SessionStorage: ...}.
package telegotd

import (
	"database/sql"
	"net"
	"os"
	"strconv"

	"github.com/go-faster/errors"
	"github.com/gotd/td/crypto"
	"github.com/gotd/td/session"

	// Чистый Go-драйвер SQLite (без cgo) — регистрируется как "sqlite".
	_ "modernc.org/sqlite"
)

// authKeyLen — длина auth_key в Telegram MTProto (256 байт).
const authKeyLen = 256

// sqliteMagic — сигнатура в начале любого SQLite-файла.
const sqliteMagic = "SQLite format 3\x00"

// FromString декодирует Telethon StringSession в *session.Data.
//
// Это тонкая обёртка над session.TelethonSession: gotd уже умеет разбирать
// строковый формат Telethon.
func FromString(s string) (*session.Data, error) {
	data, err := session.TelethonSession(s)
	if err != nil {
		return nil, errors.Wrap(err, "parse telethon string session")
	}
	return data, nil
}

// FromSQLite читает Telethon `.session` (SQLite) файл и конвертирует его в
// *session.Data.
//
// Файл открывается только на чтение (mode=ro, immutable=1), чтобы не трогать
// рабочую сессию Telethon.
func FromSQLite(path string) (*session.Data, error) {
	// mode=ro + immutable=1: открываем строго read-only, не создаём wal/shm,
	// не модифицируем исходный файл.
	dsn := "file:" + path + "?mode=ro&immutable=1"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "open sqlite")
	}
	defer func() { _ = db.Close() }()

	var (
		dcID          int
		serverAddress string
		port          int
		authKey       []byte
	)
	// Telethon хранит единственную запись авторизации в таблице sessions.
	row := db.QueryRow(`SELECT dc_id, server_address, port, auth_key FROM sessions LIMIT 1`)
	if err := row.Scan(&dcID, &serverAddress, &port, &authKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("telethon session has no authorized session row")
		}
		return nil, errors.Wrap(err, "read sessions row")
	}

	if len(authKey) != authKeyLen {
		return nil, errors.Errorf("invalid auth_key length: got %d, want %d (session not authorized?)",
			len(authKey), authKeyLen)
	}

	return buildData(dcID, serverAddress, port, authKey)
}

// Convert определяет формат входа автоматически: если input — путь к
// существующему SQLite-файлу, вызывается FromSQLite, иначе input трактуется как
// StringSession-строка.
func Convert(input string) (*session.Data, error) {
	if isSQLiteFile(input) {
		return FromSQLite(input)
	}
	return FromString(input)
}

// buildData собирает session.Data из сырых полей Telethon, вычисляя AuthKeyID.
func buildData(dcID int, serverAddress string, port int, authKey []byte) (*session.Data, error) {
	var key crypto.Key
	copy(key[:], authKey)
	id := key.WithID().ID

	return &session.Data{
		DC:        dcID,
		Addr:      net.JoinHostPort(serverAddress, strconv.Itoa(port)),
		AuthKey:   key[:],
		AuthKeyID: id[:],
		// Config и Salt намеренно пустые — gotd дотянет их при первом коннекте.
	}, nil
}

// isSQLiteFile сообщает, является ли input путём к существующему файлу с
// SQLite-сигнатурой.
func isSQLiteFile(input string) bool {
	f, err := os.Open(input)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	header := make([]byte, len(sqliteMagic))
	n, err := f.Read(header)
	if err != nil || n < len(sqliteMagic) {
		return false
	}
	return string(header) == sqliteMagic
}
