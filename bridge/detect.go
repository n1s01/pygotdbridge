package bridge

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/gotd/td/session"
)

// Kind identifies the format of a session input.
type Kind int

const (
	// KindUnknown means the input matched no known session format.
	KindUnknown Kind = iota
	// KindTelethon is a Telethon session (SQLite .session or string session).
	KindTelethon
	// KindPyrogram is a Pyrogram session (SQLite .session or string session).
	KindPyrogram
	// KindTDesktop is a Telegram Desktop tdata folder.
	KindTDesktop
	// KindGotd is a native gotd session (JSON, from session.FileStorage /
	// StorageMemory), as produced by the reverse conversion's inputs.
	KindGotd
)

func (k Kind) String() string {
	switch k {
	case KindTelethon:
		return "telethon"
	case KindPyrogram:
		return "pyrogram"
	case KindTDesktop:
		return "tdesktop"
	case KindGotd:
		return "gotd"
	default:
		return "unknown"
	}
}

// Detect reports which session format the input is: a path to a .session file,
// a path to a Telegram Desktop tdata folder, a path to a native gotd session
// file, or a raw string session (Telethon, Pyrogram, or inline gotd JSON).
//
// It returns KindUnknown when the input matches no known format. Detection
// never modifies the source and, for SQLite files, only inspects the schema.
func Detect(input string) Kind {
	if isDir(input) {
		return detectDir(input)
	}
	if isSQLiteFile(input) {
		return detectSQLite(input)
	}
	// A gotd session is JSON — either a file on disk or passed inline.
	if content, ok := readRegularFile(input); ok && isGotdSession(content) {
		return KindGotd
	}
	return detectString(input)
}

func detectDir(root string) Kind {
	if _, err := FromTDesktop(root, nil); err == nil {
		return KindTDesktop
	}
	return KindUnknown
}

func detectSQLite(path string) Kind {
	if telethon, err := sqliteHasColumn(path, "sessions", "server_address"); err == nil && telethon {
		return KindTelethon
	}
	if pyrogram, err := sqliteHasColumn(path, "sessions", "test_mode"); err == nil && pyrogram {
		return KindPyrogram
	}
	return KindUnknown
}

func detectString(s string) Kind {
	if isGotdSession([]byte(s)) {
		return KindGotd
	}
	if strings.HasPrefix(s, "1") {
		if _, err := session.TelethonSession(s); err == nil {
			return KindTelethon
		}
	}
	if _, err := FromPyrogramString(s); err == nil {
		return KindPyrogram
	}
	return KindUnknown
}

// readRegularFile reads input as a file path, returning its bytes and true only
// for existing regular files. Non-existent paths and raw string sessions (which
// are not valid paths) yield false without error.
func readRegularFile(input string) ([]byte, bool) {
	info, err := os.Stat(input)
	if err != nil || !info.Mode().IsRegular() {
		return nil, false
	}
	b, err := os.ReadFile(input)
	if err != nil {
		return nil, false
	}
	return b, true
}

// isGotdSession reports whether b is a native gotd session: the JSON envelope
// {"Version":1,"Data":{...}} carrying a full-length auth key.
func isGotdSession(b []byte) bool {
	var v struct {
		Version int
		Data    struct {
			AuthKey []byte
		}
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return false
	}
	return v.Version == 1 && len(v.Data.AuthKey) == authKeyLen
}
