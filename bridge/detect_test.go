package bridge

import (
	"path/filepath"
	"testing"
)

func TestDetectTelethon(t *testing.T) {
	want := testData(t)

	s, err := ToTelethonString(want)
	if err != nil {
		t.Fatalf("ToTelethonString: %v", err)
	}
	if got := Detect(s); got != KindTelethon {
		t.Errorf("Detect(telethon string): got %v, want %v", got, KindTelethon)
	}

	path := filepath.Join(t.TempDir(), "telethon.session")
	if err := ToTelethonSQLite(want, path); err != nil {
		t.Fatalf("ToTelethonSQLite: %v", err)
	}
	if got := Detect(path); got != KindTelethon {
		t.Errorf("Detect(telethon sqlite): got %v, want %v", got, KindTelethon)
	}
}

func TestDetectPyrogram(t *testing.T) {
	want := testData(t)
	opts := PyrogramExport{APIID: 12345, UserID: 987654321}

	s, err := ToPyrogramString(want, opts)
	if err != nil {
		t.Fatalf("ToPyrogramString: %v", err)
	}
	if got := Detect(s); got != KindPyrogram {
		t.Errorf("Detect(pyrogram string): got %v, want %v", got, KindPyrogram)
	}

	path := filepath.Join(t.TempDir(), "pyrogram.session")
	if err := ToPyrogramSQLite(want, path, opts); err != nil {
		t.Fatalf("ToPyrogramSQLite: %v", err)
	}
	if got := Detect(path); got != KindPyrogram {
		t.Errorf("Detect(pyrogram sqlite): got %v, want %v", got, KindPyrogram)
	}
}

func TestDetectUnknown(t *testing.T) {
	cases := map[string]string{
		"garbage string": "not-a-session",
		"empty":          "",
		"telethon-ish":   "1garbage",
	}
	for name, in := range cases {
		if got := Detect(in); got != KindUnknown {
			t.Errorf("Detect(%s): got %v, want %v", name, got, KindUnknown)
		}
	}
}

func TestKindString(t *testing.T) {
	cases := map[Kind]string{
		KindUnknown:  "unknown",
		KindTelethon: "telethon",
		KindPyrogram: "pyrogram",
		KindTDesktop: "tdesktop",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("Kind(%d).String(): got %q, want %q", k, got, want)
		}
	}
}
