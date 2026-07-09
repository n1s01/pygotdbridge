package bridge

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/go-faster/errors"
	"github.com/gotd/td/telegram/peers"
)

const (
	usersPrefix   = "users_"
	chatsPrefix   = "chats_"
	channelPrefix = "channel_"
	channelIDBase = 1000000000000
)

func resolvePeerKey(marked int64) (peers.Key, bool) {
	switch {
	case marked > 0:
		return peers.Key{Prefix: usersPrefix, ID: marked}, true
	case marked < 0:
		m := -marked
		if m > channelIDBase {
			return peers.Key{Prefix: channelPrefix, ID: m - channelIDBase}, true
		}
		return peers.Key{Prefix: chatsPrefix, ID: m}, true
	default:
		return peers.Key{}, false
	}
}

func MigratePeers(ctx context.Context, input string, storage peers.Storage) (int, error) {
	if ok, err := sqliteHasTable(input, "entities"); err != nil {
		return 0, err
	} else if ok {
		return migrateTelethonPeers(ctx, input, storage)
	}
	if ok, err := sqliteHasTable(input, "peers"); err != nil {
		return 0, err
	} else if ok {
		return migratePyrogramPeers(ctx, input, storage)
	}
	return 0, errors.New("no peer table (entities/peers) found")
}

func MigratePeersToMemory(ctx context.Context, input string) (*peers.InmemoryStorage, int, error) {
	storage := &peers.InmemoryStorage{}
	n, err := MigratePeers(ctx, input, storage)
	if err != nil {
		return nil, 0, err
	}
	return storage, n, nil
}

func migrateTelethonPeers(ctx context.Context, path string, storage peers.Storage) (int, error) {
	db, err := openSQLiteRO(path)
	if err != nil {
		return 0, errors.Wrap(err, "open sqlite")
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(ctx, `SELECT id, hash, phone FROM entities`)
	if err != nil {
		return 0, errors.Wrap(err, "query entities")
	}
	defer func() { _ = rows.Close() }()

	return saveRows(ctx, rows, storage, func(phone sql.NullInt64) string {
		if !phone.Valid || phone.Int64 == 0 {
			return ""
		}
		return strconv.FormatInt(phone.Int64, 10)
	})
}

func migratePyrogramPeers(ctx context.Context, path string, storage peers.Storage) (int, error) {
	db, err := openSQLiteRO(path)
	if err != nil {
		return 0, errors.Wrap(err, "open sqlite")
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(ctx, `SELECT id, access_hash, phone_number FROM peers`)
	if err != nil {
		return 0, errors.Wrap(err, "query peers")
	}
	defer func() { _ = rows.Close() }()

	var (
		count int
		id    int64
		hash  sql.NullInt64
		phone sql.NullString
	)
	for rows.Next() {
		if err := rows.Scan(&id, &hash, &phone); err != nil {
			return count, errors.Wrap(err, "scan peer")
		}
		key, ok := resolvePeerKey(id)
		if !ok {
			continue
		}
		if err := storage.Save(ctx, key, peers.Value{AccessHash: hash.Int64}); err != nil {
			return count, errors.Wrap(err, "save peer")
		}
		if phone.Valid && phone.String != "" {
			if err := storage.SavePhone(ctx, phone.String, key); err != nil {
				return count, errors.Wrap(err, "save phone")
			}
		}
		count++
	}
	return count, rows.Err()
}

func saveRows(ctx context.Context, rows *sql.Rows, storage peers.Storage, phoneStr func(sql.NullInt64) string) (int, error) {
	var (
		count int
		id    int64
		hash  sql.NullInt64
		phone sql.NullInt64
	)
	for rows.Next() {
		if err := rows.Scan(&id, &hash, &phone); err != nil {
			return count, errors.Wrap(err, "scan entity")
		}
		key, ok := resolvePeerKey(id)
		if !ok {
			continue
		}
		if err := storage.Save(ctx, key, peers.Value{AccessHash: hash.Int64}); err != nil {
			return count, errors.Wrap(err, "save peer")
		}
		if p := phoneStr(phone); p != "" {
			if err := storage.SavePhone(ctx, p, key); err != nil {
				return count, errors.Wrap(err, "save phone")
			}
		}
		count++
	}
	return count, rows.Err()
}

func sqliteHasTable(path, table string) (bool, error) {
	db, err := openSQLiteRO(path)
	if err != nil {
		return false, errors.Wrap(err, "open sqlite")
	}
	defer func() { _ = db.Close() }()

	var name string
	err = db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
	).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "query sqlite_master")
	}
	return true, nil
}
