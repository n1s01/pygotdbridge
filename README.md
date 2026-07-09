# pygotdbridge

Use existing **Telethon** and **Pyrogram** sessions with [gotd/td](https://github.com/gotd/td) — no re-login required.

It converts a third-party session into a native gotd `session.Storage` that plugs straight into `telegram.Client`.

| Library  | SQLite `.session` | String session |
|----------|:---:|:---:|
| Telethon | ✅ | ✅ |
| Pyrogram | ✅ | ✅ |

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
| `Convert(input string) (*session.Data, error)` | Auto-detect format → `session.Data`. |
| `Storage(data *session.Data) (*session.StorageMemory, error)` | `session.Data` → `session.Storage`. |
| `FromTelethonString` / `FromTelethonSQLite` | Telethon only. |
| `FromPyrogramString` / `FromPyrogramSQLite` | Pyrogram only. |

## Notes

- `app_id` / `app_hash` are still required for gotd's `initConnection` (the auth key is account-bound, not app-bound).
- Session files are opened read-only; the source is never modified.
- Pyrogram stores only `dc_id` (no address); it is mapped to the fixed Telegram DC IPs.
