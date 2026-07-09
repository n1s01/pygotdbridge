// Package gotdbridge конвертирует существующие сессии сторонних Telegram-библиотек
// (Telethon, Pyrogram) в нативный формат gotd (github.com/gotd/td), позволяя
// работать аккаунтом через gotd без повторной авторизации.
//
// Поддерживаются форматы:
//
//   - Telethon: SQLite `.session` файл и StringSession-строка;
//   - Pyrogram: SQLite `.session` файл и string session.
//
// Основная точка входа — StorageFromInput: на вход даётся сессия, на выход —
// готовый session.Storage для telegram.Options{SessionStorage: ...}.
package gotdbridge

import (
	"database/sql"
	"os"

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

// Convert определяет формат входа автоматически и конвертирует его в
// *session.Data. Если input — путь к существующему SQLite-файлу, распознаётся
// схема (Telethon/Pyrogram); иначе input трактуется как string session.
func Convert(input string) (*session.Data, error) {
	if isSQLiteFile(input) {
		return FromTelethonSQLite(input)
	}
	return FromTelethonString(input)
}

// buildData собирает session.Data из общих полей (dc_id, адрес DC, auth_key),
// вычисляя AuthKeyID. Config и Salt остаются пустыми — gotd дотянет их сам.
func buildData(dcID int, addr string, authKey []byte) (*session.Data, error) {
	if len(authKey) != authKeyLen {
		return nil, errors.Errorf("invalid auth_key length: got %d, want %d (session not authorized?)",
			len(authKey), authKeyLen)
	}
	var key crypto.Key
	copy(key[:], authKey)
	id := key.WithID().ID

	return &session.Data{
		DC:        dcID,
		Addr:      addr,
		AuthKey:   key[:],
		AuthKeyID: id[:],
	}, nil
}

// openSQLiteRO открывает SQLite-файл строго на чтение (mode=ro, immutable=1),
// чтобы не модифицировать исходную сессию сторонней библиотеки.
func openSQLiteRO(path string) (*sql.DB, error) {
	return sql.Open("sqlite", "file:"+path+"?mode=ro&immutable=1")
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
