# pygotdbridge

Мост, который позволяет использовать существующие сессии **Telethon** и
**Pyrogram** через **[gotd/td](https://github.com/gotd/td)** — без повторной
авторизации аккаунта.

Поддерживаемые форматы входа (детектятся автоматически):

| Библиотека | SQLite `.session` | String session |
|---|---|---|
| **Telethon** | ✅ | ✅ |
| **Pyrogram** | ✅ | ✅ (3 формата) |

На выходе — готовый `session.Storage`, который напрямую втыкается в
`telegram.Client`.

## Установка

```bash
go get github.com/n1s01/pygotdbridge
```

Зависимости: `github.com/gotd/td` и `modernc.org/sqlite` (чистый Go SQLite-драйвер,
без cgo — легко кросс-компилится).

## Использование

Основная функция — «воркер» `StorageFromInput`: принимает сессию, возвращает
`session.Storage` для gotd.

```go
import (
    "github.com/gotd/td/telegram"
    "github.com/n1s01/pygotdbridge"
)

// input — путь к .session файлу ЛИБО StringSession-строка (формат детектится сам).
st, err := pygotdbridge.StorageFromInput(input)
if err != nil {
    log.Fatal(err)
}

client := telegram.NewClient(appID, appHash, telegram.Options{
    SessionStorage: st,
})

client.Run(ctx, func(ctx context.Context) error {
    self, err := client.Self(ctx)
    // ... работаем аккаунтом через gotd
    return err
})
```

## API

| Функция | Назначение |
|---|---|
| `StorageFromInput(input string) (*session.StorageMemory, error)` | Сессия → готовый `session.Storage`. Главная точка входа. |
| `Convert(input string) (*session.Data, error)` | Авто-детект формата → `session.Data`. |
| `FromTelethonString(s string) (*session.Data, error)` | Только Telethon StringSession. |
| `FromTelethonSQLite(path string) (*session.Data, error)` | Только Telethon `.session` (SQLite). |
| `FromPyrogramString(s string) (*session.Data, error)` | Только Pyrogram string session. |
| `FromPyrogramSQLite(path string) (*session.Data, error)` | Только Pyrogram `.session` (SQLite). |
| `Storage(data *session.Data) (*session.StorageMemory, error)` | `session.Data` → `session.Storage`. |

## Демо

```bash
APP_ID=123456 APP_HASH=abcdef... go run ./cmd/demo /path/to/account.session
# или
APP_ID=123456 APP_HASH=abcdef... go run ./cmd/demo "1BQANOTEu...строка"
```

Печатает `id / first name / username` текущего аккаунта — подтверждение, что auth
key принят Telegram без переавторизации.

## Важные нюансы

- **AppID / AppHash всё равно нужны.** Auth key привязан к аккаунту и DC, а не к
  приложению, но gotd отправляет `api_id` в `initConnection`. Рекомендуется
  использовать те же `api_id/api_hash`, что и Telethon; технически подойдёт любая
  валидная пара.
- **Файл сессии открывается только на чтение** (`mode=ro&immutable=1`) — исходная
  сессия не модифицируется.
- **Pyrogram не хранит адрес DC** в сессии (только `dc_id`) — адрес
  восстанавливается по фиксированной таблице продакшн/тестовых дата-центров.
- **Salt и Config не переносятся** — gotd дотягивает их сам при первом коннекте.
- Неавторизованная сессия (пустой/короткий `auth_key`) → внятная ошибка.
