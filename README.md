# pygotdbridge

Use existing **Telethon**, **Pyrogram**, and **Telegram Desktop** sessions with [gotd/td](https://github.com/gotd/td) — no re-login required.

It converts a third-party session into a native gotd `session.Storage` that plugs straight into `telegram.Client`, and back — export a gotd session as a Telethon/Pyrogram `.session` file or string.

| Source | SQLite `.session` | String session | `tdata` folder |
|----------|:---:|:---:|:---:|
| Telethon | ✅ | ✅ | — |
| Pyrogram | ✅ | ✅ | — |
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

opts := bridge.PyrogramExport{APIID: apiID, UserID: userID}
ps, _ := bridge.ToPyrogramString(data, opts)
_ = bridge.ToPyrogramSQLite(data, "pyrogram.session", opts)
```

| Function | Description |
|----------|-------------|
| `ToTelethonString(data) (string, error)` | gotd → Telethon string session. |
| `ToTelethonSQLite(data, path) error` | gotd → Telethon `.session` file. |
| `ToPyrogramString(data, PyrogramExport) (string, error)` | gotd → Pyrogram string session. |
| `ToPyrogramSQLite(data, path, PyrogramExport) error` | gotd → Pyrogram `.session` file. |

`PyrogramExport` carries fields absent from `session.Data` (`APIID`, `TestMode`, `UserID`, `IsBot`). SQLite exports overwrite the target path.

## Notes

- `app_id` / `app_hash` are still required for gotd's `initConnection` (the auth key is account-bound, not app-bound).
- Session files are opened read-only; the source is never modified.
- Pyrogram stores only `dc_id` (no address); it is mapped to the fixed Telegram DC IPs.
