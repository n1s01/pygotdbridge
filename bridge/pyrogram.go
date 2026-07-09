package bridge

import (
	"database/sql"
	"encoding/base64"
	"strings"

	"github.com/go-faster/errors"
	"github.com/gotd/td/session"
)

func FromPyrogramSQLite(path string) (*session.Data, error) {
	db, err := openSQLiteRO(path)
	if err != nil {
		return nil, errors.Wrap(err, "open sqlite")
	}
	defer func() { _ = db.Close() }()

	var (
		dcID     int
		testMode int
		authKey  []byte
	)
	row := db.QueryRow(`SELECT dc_id, test_mode, auth_key FROM sessions LIMIT 1`)
	if err := row.Scan(&dcID, &testMode, &authKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("pyrogram session has no session row")
		}
		return nil, errors.Wrap(err, "read sessions row")
	}

	addr, err := dcAddr(dcID, testMode != 0)
	if err != nil {
		return nil, err
	}
	return buildData(dcID, addr, authKey)
}

func FromPyrogramString(s string) (*session.Data, error) {
	data, err := base64.URLEncoding.DecodeString(padBase64(s))
	if err != nil {
		return nil, errors.Wrap(err, "decode base64")
	}

	var (
		dcID    int
		test    bool
		authKey []byte
	)
	switch len(data) {
	case 271:
		dcID = int(data[0])
		test = data[5] != 0
		authKey = data[6:262]
	case 267:
		dcID = int(data[0])
		test = data[1] != 0
		authKey = data[2:258]
	case 263:
		dcID = int(data[0])
		test = data[1] != 0
		authKey = data[2:258]
	default:
		return nil, errors.Errorf("unexpected pyrogram session length: %d", len(data))
	}

	addr, err := dcAddr(dcID, test)
	if err != nil {
		return nil, err
	}
	return buildData(dcID, addr, authKey)
}

func padBase64(s string) string {
	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}
	return s
}
