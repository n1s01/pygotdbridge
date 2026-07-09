package bridge

import (
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/go-faster/errors"
	"github.com/gotd/td/session"
)

const (
	telethonVersion = 8
	pyrogramVersion = 3
)

type PyrogramExport struct {
	APIID    int
	TestMode bool
	UserID   int64
	IsBot    bool
}

func ToTelethonString(data *session.Data) (string, error) {
	if err := validateExport(data); err != nil {
		return "", err
	}
	ip, port, err := splitIPPort(data.Addr)
	if err != nil {
		return "", err
	}
	ipb := ip.To4()
	if ipb == nil {
		ipb = ip.To16()
	}

	buf := make([]byte, 0, 1+len(ipb)+2+authKeyLen)
	buf = append(buf, byte(data.DC))
	buf = append(buf, ipb...)
	buf = binary.BigEndian.AppendUint16(buf, port)
	buf = append(buf, data.AuthKey...)

	return "1" + base64.URLEncoding.EncodeToString(buf), nil
}

func ToPyrogramString(data *session.Data, opts PyrogramExport) (string, error) {
	if err := validateExport(data); err != nil {
		return "", err
	}

	buf := make([]byte, 0, 271)
	buf = append(buf, byte(data.DC))
	buf = binary.BigEndian.AppendUint32(buf, uint32(opts.APIID))
	buf = append(buf, boolByte(opts.TestMode))
	buf = append(buf, data.AuthKey...)
	buf = binary.BigEndian.AppendUint64(buf, uint64(opts.UserID))
	buf = append(buf, boolByte(opts.IsBot))

	return strings.TrimRight(base64.URLEncoding.EncodeToString(buf), "="), nil
}

func ToTelethonSQLite(data *session.Data, path string) error {
	if err := validateExport(data); err != nil {
		return err
	}
	host, portStr, err := net.SplitHostPort(data.Addr)
	if err != nil {
		return errors.Wrap(err, "split addr")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return errors.Wrap(err, "parse port")
	}

	return writeSQLite(path, telethonSchema, func(db *sql.DB) error {
		if _, err := db.Exec(`INSERT INTO version VALUES (?)`, telethonVersion); err != nil {
			return errors.Wrap(err, "insert version")
		}
		if _, err := db.Exec(
			`INSERT OR REPLACE INTO sessions VALUES (?, ?, ?, ?, ?, ?)`,
			data.DC, host, port, data.AuthKey, nil, []byte{},
		); err != nil {
			return errors.Wrap(err, "insert session")
		}
		return nil
	})
}

func ToPyrogramSQLite(data *session.Data, path string, opts PyrogramExport) error {
	if err := validateExport(data); err != nil {
		return err
	}

	return writeSQLite(path, pyrogramSchema, func(db *sql.DB) error {
		if _, err := db.Exec(`INSERT INTO version VALUES (?)`, pyrogramVersion); err != nil {
			return errors.Wrap(err, "insert version")
		}
		if _, err := db.Exec(
			`INSERT INTO sessions VALUES (?, ?, ?, ?, ?, ?, ?)`,
			data.DC, opts.APIID, boolInt(opts.TestMode), data.AuthKey, 0, opts.UserID, boolInt(opts.IsBot),
		); err != nil {
			return errors.Wrap(err, "insert session")
		}
		return nil
	})
}

func writeSQLite(path, schema string, seed func(*sql.DB) error) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "remove existing file")
	}
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		return errors.Wrap(err, "open sqlite")
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(schema); err != nil {
		return errors.Wrap(err, "create schema")
	}
	return seed(db)
}

func validateExport(data *session.Data) error {
	if data == nil {
		return errors.New("nil session data")
	}
	if len(data.AuthKey) != authKeyLen {
		return errors.Errorf("invalid auth_key length: got %d, want %d", len(data.AuthKey), authKeyLen)
	}
	return nil
}

func splitIPPort(addr string) (net.IP, uint16, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, 0, errors.Wrap(err, "split addr")
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, 0, errors.Errorf("addr host is not an IP: %q", host)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, 0, errors.Wrap(err, "parse port")
	}
	return ip, uint16(port), nil
}

func boolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

const telethonSchema = `
CREATE TABLE version (version integer primary key);
CREATE TABLE sessions (
	dc_id integer primary key,
	server_address text,
	port integer,
	auth_key blob,
	takeout_id integer,
	tmp_auth_key blob
);
CREATE TABLE entities (
	id integer primary key,
	hash integer not null,
	username text,
	phone integer,
	name text,
	date integer
);
CREATE TABLE sent_files (
	md5_digest blob,
	file_size integer,
	type integer,
	id integer,
	hash integer,
	primary key(md5_digest, file_size, type)
);
CREATE TABLE update_state (
	id integer primary key,
	pts integer,
	qts integer,
	date integer,
	seq integer
);
`

const pyrogramSchema = `
CREATE TABLE sessions (
	dc_id     INTEGER PRIMARY KEY,
	api_id    INTEGER,
	test_mode INTEGER,
	auth_key  BLOB,
	date      INTEGER NOT NULL,
	user_id   INTEGER,
	is_bot    INTEGER
);
CREATE TABLE peers (
	id             INTEGER PRIMARY KEY,
	access_hash    INTEGER,
	type           INTEGER NOT NULL,
	username       TEXT,
	phone_number   TEXT,
	last_update_on INTEGER NOT NULL DEFAULT (CAST(STRFTIME('%s', 'now') AS INTEGER))
);
CREATE TABLE version (
	number INTEGER PRIMARY KEY
);
CREATE INDEX idx_peers_id ON peers (id);
CREATE INDEX idx_peers_username ON peers (username);
CREATE INDEX idx_peers_phone_number ON peers (phone_number);
CREATE TRIGGER trg_peers_last_update_on AFTER UPDATE ON peers
BEGIN
	UPDATE peers SET last_update_on = CAST(STRFTIME('%s', 'now') AS INTEGER) WHERE id = NEW.id;
END;
`
