package bridge

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/gotd/td/telegram/peers"
)

func execAll(t *testing.T, path string, stmts ...string) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}
}

func assertHash(t *testing.T, st *peers.InmemoryStorage, prefix string, id, want int64) {
	t.Helper()
	v, found, err := st.Find(context.Background(), peers.Key{Prefix: prefix, ID: id})
	if err != nil {
		t.Fatalf("find %s%d: %v", prefix, id, err)
	}
	if !found {
		t.Fatalf("key %s%d not found", prefix, id)
	}
	if v.AccessHash != want {
		t.Errorf("%s%d access_hash: got %d, want %d", prefix, id, v.AccessHash, want)
	}
}

const channelMarked = -(channelIDBase + 789)

func TestResolvePeerKey(t *testing.T) {
	cases := []struct {
		marked int64
		prefix string
		id     int64
	}{
		{123, usersPrefix, 123},
		{-456, chatsPrefix, 456},
		{channelMarked, channelPrefix, 789},
	}
	for _, c := range cases {
		key, ok := resolvePeerKey(c.marked)
		if !ok || key.Prefix != c.prefix || key.ID != c.id {
			t.Errorf("resolvePeerKey(%d) = %+v ok=%v, want {%s %d}", c.marked, key, ok, c.prefix, c.id)
		}
	}
	if _, ok := resolvePeerKey(0); ok {
		t.Error("resolvePeerKey(0) should be invalid")
	}
}

func TestMigrateTelethonPeers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "telethon.session")
	execAll(t, path,
		`CREATE TABLE entities (id integer primary key, hash integer not null, username text, phone integer, name text, date integer)`,
		`INSERT INTO entities VALUES (123, 111, 'alice', 15551234567, 'Alice', 0)`,
		`INSERT INTO entities VALUES (-456, 0, NULL, NULL, 'Group', 0)`,
		`INSERT INTO entities VALUES (-1000000000789, 222, 'chan', NULL, 'Channel', 0)`,
	)

	st, n, err := MigratePeersToMemory(context.Background(), path)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if n != 3 {
		t.Errorf("count: got %d, want 3", n)
	}
	assertHash(t, st, usersPrefix, 123, 111)
	assertHash(t, st, chatsPrefix, 456, 0)
	assertHash(t, st, channelPrefix, 789, 222)

	key, _, found, err := st.FindPhone(context.Background(), "15551234567")
	if err != nil || !found {
		t.Fatalf("find phone: found=%v err=%v", found, err)
	}
	if key.Prefix != usersPrefix || key.ID != 123 {
		t.Errorf("phone key: got %+v, want users_123", key)
	}
}

func TestMigratePyrogramPeers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pyrogram.session")
	execAll(t, path,
		`CREATE TABLE peers (id integer primary key, access_hash integer, type text not null, phone_number text, last_update_on integer)`,
		`INSERT INTO peers VALUES (123, 111, 'user', '15551234567', 0)`,
		`INSERT INTO peers VALUES (-456, NULL, 'group', NULL, 0)`,
		`INSERT INTO peers VALUES (-1000000000789, 222, 'channel', NULL, 0)`,
	)

	st, n, err := MigratePeersToMemory(context.Background(), path)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if n != 3 {
		t.Errorf("count: got %d, want 3", n)
	}
	assertHash(t, st, usersPrefix, 123, 111)
	assertHash(t, st, chatsPrefix, 456, 0)
	assertHash(t, st, channelPrefix, 789, 222)

	key, _, found, err := st.FindPhone(context.Background(), "15551234567")
	if err != nil || !found {
		t.Fatalf("find phone: found=%v err=%v", found, err)
	}
	if key.Prefix != usersPrefix || key.ID != 123 {
		t.Errorf("phone key: got %+v, want users_123", key)
	}
}

func TestMigratePeersNoTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.session")
	execAll(t, path, `CREATE TABLE sessions (dc_id integer)`)
	if _, err := MigratePeers(context.Background(), path, &peers.InmemoryStorage{}); err == nil {
		t.Error("expected error when no peer table present")
	}
}
