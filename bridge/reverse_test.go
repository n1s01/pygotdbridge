package bridge

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/gotd/td/session"
)

func testData(t *testing.T) *session.Data {
	t.Helper()
	key := make([]byte, authKeyLen)
	for i := range key {
		key[i] = byte(i * 7)
	}
	data, err := buildData(2, "149.154.167.51:443", key)
	if err != nil {
		t.Fatalf("buildData: %v", err)
	}
	return data
}

func assertSame(t *testing.T, want, got *session.Data) {
	t.Helper()
	if got.DC != want.DC {
		t.Errorf("DC: got %d, want %d", got.DC, want.DC)
	}
	if got.Addr != want.Addr {
		t.Errorf("Addr: got %q, want %q", got.Addr, want.Addr)
	}
	if !bytes.Equal(got.AuthKey, want.AuthKey) {
		t.Errorf("AuthKey mismatch")
	}
	if !bytes.Equal(got.AuthKeyID, want.AuthKeyID) {
		t.Errorf("AuthKeyID mismatch")
	}
}

func TestTelethonStringRoundTrip(t *testing.T) {
	want := testData(t)
	s, err := ToTelethonString(want)
	if err != nil {
		t.Fatalf("ToTelethonString: %v", err)
	}
	got, err := FromTelethonString(s)
	if err != nil {
		t.Fatalf("FromTelethonString: %v", err)
	}
	assertSame(t, want, got)
}

func TestPyrogramStringRoundTrip(t *testing.T) {
	want := testData(t)
	s, err := ToPyrogramString(want, PyrogramExport{APIID: 12345, UserID: 987654321})
	if err != nil {
		t.Fatalf("ToPyrogramString: %v", err)
	}
	got, err := FromPyrogramString(s)
	if err != nil {
		t.Fatalf("FromPyrogramString: %v", err)
	}
	assertSame(t, want, got)
}

func TestTelethonSQLiteRoundTrip(t *testing.T) {
	want := testData(t)
	path := filepath.Join(t.TempDir(), "telethon.session")
	if err := ToTelethonSQLite(want, path); err != nil {
		t.Fatalf("ToTelethonSQLite: %v", err)
	}
	got, err := FromTelethonSQLite(path)
	if err != nil {
		t.Fatalf("FromTelethonSQLite: %v", err)
	}
	assertSame(t, want, got)

	auto, err := Convert(path)
	if err != nil {
		t.Fatalf("Convert(telethon sqlite): %v", err)
	}
	assertSame(t, want, auto)
}

func TestPyrogramSQLiteRoundTrip(t *testing.T) {
	want := testData(t)
	path := filepath.Join(t.TempDir(), "pyrogram.session")
	if err := ToPyrogramSQLite(want, path, PyrogramExport{APIID: 12345, UserID: 987654321}); err != nil {
		t.Fatalf("ToPyrogramSQLite: %v", err)
	}
	got, err := FromPyrogramSQLite(path)
	if err != nil {
		t.Fatalf("FromPyrogramSQLite: %v", err)
	}
	assertSame(t, want, got)

	auto, err := Convert(path)
	if err != nil {
		t.Fatalf("Convert(pyrogram sqlite): %v", err)
	}
	assertSame(t, want, auto)
}

func TestExportRejectsBadAuthKey(t *testing.T) {
	bad := &session.Data{DC: 2, Addr: "149.154.167.51:443", AuthKey: []byte{1, 2, 3}}
	if _, err := ToTelethonString(bad); err == nil {
		t.Error("expected error for short auth key")
	}
	if _, err := ToPyrogramString(bad, PyrogramExport{}); err == nil {
		t.Error("expected error for short auth key")
	}
}
