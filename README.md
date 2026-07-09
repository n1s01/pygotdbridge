# pygotdbridge

Use existing **Telethon**, **Pyrogram**, and **Telegram Desktop** sessions with [gotd/td](https://github.com/gotd/td) — no re-login required.

It converts a third-party session into a native gotd `session.Storage` that plugs straight into `telegram.Client`, and back — export a gotd session as a Telethon/Pyrogram `.session` file or string.

| Source | SQLite `.session` | String session | `tdata` folder |
|----------|:---:|:---:|:---:|
| Telethon | ✅ | ✅ | ✅ |
| Pyrogram | ✅ | ✅ | ✅ |
| Telegram Desktop | — | — | ✅ |

## Install

```bash
go get github.com/n1s01/pygotdbridge/bridge
```

## Usage

```go
import (
	"github.com/gotd/td/telegram"
	"github.com/n1s01/pygotdbridge/bridge"
)

// input: path to a .session file or a string session (auto-detected).
st, err := bridge.StorageFromInput(input)
if err != nil {
	log.Fatal(err)
}

client := telegram.NewClient(appID, appHash, telegram.Options{
	SessionStorage: st,
})
```

## API

| Function | Description |
|----------|-------------|
| `StorageFromInput(input string) (*session.StorageMemory, error)` | Session → ready `session.Storage`. Main entry point. |
| `Convert(input string) (*session.Data, error)` | Auto-detect format → `session.Data`. A directory is read as a Telegram Desktop `tdata` folder. |
| `Detect(input string) Kind` | Report the session format without converting: `KindTelethon`, `KindPyrogram`, `KindTDesktop`, or `KindUnknown`. |
| `FromTDesktop(root string, passcode []byte) (*session.Data, error)` | Telegram Desktop `tdata` → `session.Data` (first account). |
| `FromTDesktopAll(root string, passcode []byte) ([]*session.Data, error)` | All accounts in a `tdata` folder. |
| `StorageFromTDesktop(root string, passcode []byte) (*session.StorageMemory, error)` | `tdata` → ready `session.Storage`. |
| `Storage(data *session.Data) (*session.StorageMemory, error)` | `session.Data` → `session.Storage`. |
| `FromTelethonString` / `FromTelethonSQLite` | Telethon only. |
| `FromPyrogramString` / `FromPyrogramSQLite` | Pyrogram only. |

## Reverse conversion

Export a gotd `session.Data` back to a Telethon/Pyrogram session:

```go
data, _ := bridge.Convert(input)

ts, _ := bridge.ToTelethonString(data)
_ = bridge.ToTelethonSQLite(data, "telethon.session")

// PyrogramExport is optional — omit it for a session without api_id/user_id.
ps, _ := bridge.ToPyrogramString(data)
_ = bridge.ToPyrogramSQLite(data, "pyrogram.session")

// ...or supply the extra fields Pyrogram stores:
opts := bridge.PyrogramExport{APIID: apiID, UserID: userID}
ps, _ = bridge.ToPyrogramString(data, opts)
```

| Function | Description |
|----------|-------------|
| `ToTelethonString(data) (string, error)` | gotd → Telethon string session. |
| `ToTelethonSQLite(data, path) error` | gotd → Telethon `.session` file. |
| `ToPyrogramString(data, ...PyrogramExport) (string, error)` | gotd → Pyrogram string session. |
| `ToPyrogramSQLite(data, path, ...PyrogramExport) error` | gotd → Pyrogram `.session` file. |
| `ToTDesktopFiles(data, userID, passcode) (TDesktopFiles, error)` | gotd → tdata files, in memory. |
| `ToTDesktop(data, userID, root, passcode) error` | gotd → tdata folder, written to `root`. |

`PyrogramExport` is optional and carries fields absent from `session.Data` (`APIID`,
`TestMode`, `UserID`, `IsBot`). Without it the session still holds a valid auth key —
Pyrogram just needs `api_id` passed to `Client` at load time. SQLite exports overwrite the target path.

`tdata` embeds the account's Telegram user ID alongside the DC/auth key, so `ToTDesktop`
needs it as an explicit argument — pass `0` if you don't know it; Telegram Desktop will
still connect, since the ID is only used for its own bookkeeping.

`ToTDesktopFiles` returns a `TDesktopFiles` (a `map[string][]byte` of tdata-root-relative
file names to contents) instead of writing straight to disk — `ToTDesktop` is a thin
wrapper around it for the common "give me a folder" case. Use the raw map form to zip the
result, ship it over the network, or otherwise decide storage yourself:

```go
files, _ := bridge.ToTDesktopFiles(data, userID, nil) // no local passcode
for name, content := range files {
	_ = os.WriteFile(filepath.Join(root, name), content, 0o600)
}
```

## Peer cache migration

Telethon (`entities`) and Pyrogram (`peers`) SQLite files store an `id → access_hash`
table. Porting it into gotd's peer storage lets `telegram/peers` resolve users, chats,
and channels offline — without re-resolving each one (fewer requests, less flood risk).

```go
import "github.com/gotd/td/telegram/peers"

storage, n, err := bridge.MigratePeersToMemory(ctx, "account.session")
// n = number of peers migrated

mgr := peers.Options{Storage: storage}.Build(api)
user, err := mgr.ResolveUserID(ctx, userID) // served from cache
```

| Function | Description |
|----------|-------------|
| `MigratePeers(ctx, input, peers.Storage) (int, error)` | Port `access_hash`es into an existing peer storage. |
| `MigratePeersToMemory(ctx, input) (*peers.InmemoryStorage, int, error)` | Migrate into a fresh in-memory storage. |

Telethon and Pyrogram both use bot-API marked IDs, so a single decoder maps them to
gotd's `users_` / `chats_` / `channel_` keys. Phone numbers are migrated too.

## Notes

- `app_id` / `app_hash` are still required for gotd's `initConnection` (the auth key is account-bound, not app-bound).
- Session files are opened read-only; the source is never modified.
- Pyrogram stores only `dc_id` (no address); it is mapped to the fixed Telegram DC IPs.
