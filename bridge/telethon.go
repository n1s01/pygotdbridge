package bridge

import (
	"database/sql"
	"net"
	"strconv"

	"github.com/go-faster/errors"
	"github.com/gotd/td/session"
)

// FromTelethonString декодирует Telethon StringSession в *session.Data.
//
// Это тонкая обёртка над session.TelethonSession: gotd уже умеет разбирать
// строковый формат Telethon.
func FromTelethonString(s string) (*session.Data, error) {
	data, err := session.TelethonSession(s)
	if err != nil {
		return nil, errors.Wrap(err, "parse telethon string session")
	}
	return data, nil
}

// FromTelethonSQLite читает Telethon `.session` (SQLite) файл и конвертирует его
// в *session.Data. Файл открывается только на чтение.
//
// Telethon хранит адрес DC прямо в таблице sessions (server_address, port).
func FromTelethonSQLite(path string) (*session.Data, error) {
	db, err := openSQLiteRO(path)
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
	row := db.QueryRow(`SELECT dc_id, server_address, port, auth_key FROM sessions LIMIT 1`)
	if err := row.Scan(&dcID, &serverAddress, &port, &authKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("telethon session has no authorized session row")
		}
		return nil, errors.Wrap(err, "read sessions row")
	}

	return buildData(dcID, net.JoinHostPort(serverAddress, strconv.Itoa(port)), authKey)
}
