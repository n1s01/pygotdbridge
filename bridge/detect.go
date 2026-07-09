package bridge

import (
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
)

func (k Kind) String() string {
	switch k {
	case KindTelethon:
		return "telethon"
	case KindPyrogram:
		return "pyrogram"
	case KindTDesktop:
		return "tdesktop"
	default:
		return "unknown"
	}
}

// Detect reports which session format the input is: a path to a .session file,
// a path to a Telegram Desktop tdata folder, or a raw string session.
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
