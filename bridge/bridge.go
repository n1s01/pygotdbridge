package bridge

import (
	"database/sql"
	"os"
	"strings"

	"github.com/go-faster/errors"
	"github.com/gotd/td/crypto"
	"github.com/gotd/td/session"

	_ "modernc.org/sqlite"
)

const authKeyLen = 256

const sqliteMagic = "SQLite format 3\x00"

func Convert(input string) (*session.Data, error) {
	if isSQLiteFile(input) {
		return fromSQLiteAuto(input)
	}
	return fromStringAuto(input)
}

func fromSQLiteAuto(path string) (*session.Data, error) {
	telethon, err := sqliteHasColumn(path, "sessions", "server_address")
	if err != nil {
		return nil, err
	}
	if telethon {
		return FromTelethonSQLite(path)
	}
	return FromPyrogramSQLite(path)
}

func fromStringAuto(s string) (*session.Data, error) {
	if strings.HasPrefix(s, "1") {
		if data, err := FromTelethonString(s); err == nil {
			return data, nil
		}
	}
	return FromPyrogramString(s)
}

func sqliteHasColumn(path, table, column string) (bool, error) {
	db, err := openSQLiteRO(path)
	if err != nil {
		return false, errors.Wrap(err, "open sqlite")
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false, errors.Wrap(err, "table_info")
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, errors.Wrap(err, "scan column")
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

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

func openSQLiteRO(path string) (*sql.DB, error) {
	return sql.Open("sqlite", "file:"+path+"?mode=ro&immutable=1")
}

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
