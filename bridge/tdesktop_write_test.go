package bridge

import (
	"testing"

	"github.com/gotd/td/session"
)

func TestToTDesktopRoundTrip(t *testing.T) {
	want := testData(t)
	const userID = int64(245322359)

	dir := t.TempDir()
	if err := ToTDesktop(want, userID, dir, nil); err != nil {
		t.Fatalf("ToTDesktop: %v", err)
	}

	got, err := FromTDesktop(dir, nil)
	if err != nil {
		t.Fatalf("FromTDesktop: %v", err)
	}
	// tdata never stores a server address, only the DC ID: gotd resolves
	// Addr itself from its own DC table, which need not match ours.
	if got.DC != want.DC {
		t.Errorf("DC: got %d, want %d", got.DC, want.DC)
	}
	if string(got.AuthKey) != string(want.AuthKey) {
		t.Errorf("AuthKey mismatch")
	}
}

func TestToTDesktopRoundTripWithPasscode(t *testing.T) {
	want := testData(t)
	passcode := []byte("hunter2")

	dir := t.TempDir()
	if err := ToTDesktop(want, 123, dir, passcode); err != nil {
		t.Fatalf("ToTDesktop: %v", err)
	}

	if _, err := FromTDesktop(dir, nil); err == nil {
		t.Fatal("expected error reading passcode-protected tdata without a passcode")
	}

	got, err := FromTDesktop(dir, passcode)
	if err != nil {
		t.Fatalf("FromTDesktop: %v", err)
	}
	if got.DC != want.DC {
		t.Errorf("DC: got %d, want %d", got.DC, want.DC)
	}
	if string(got.AuthKey) != string(want.AuthKey) {
		t.Errorf("AuthKey mismatch")
	}
}

func TestToTDesktopFilesRejectsBadAuthKey(t *testing.T) {
	bad := &session.Data{DC: 2, Addr: "149.154.167.51:443", AuthKey: []byte{1, 2, 3}}
	if _, err := ToTDesktopFiles(bad, 1, nil); err == nil {
		t.Error("expected error for short auth key")
	}
}
