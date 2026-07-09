package gotdbridge

import (
	"context"

	"github.com/go-faster/errors"
	"github.com/gotd/td/session"
)

// Storage упаковывает *session.Data в готовый *session.StorageMemory, который
// реализует интерфейс session.Storage и передаётся в
// telegram.Options{SessionStorage: ...}.
//
// Данные записываются в том же JSON-конверте, что использует gotd (через
// session.Loader), поэтому клиент прочитает их штатно.
func Storage(data *session.Data) (*session.StorageMemory, error) {
	if data == nil {
		return nil, errors.New("nil session data")
	}
	mem := &session.StorageMemory{}
	loader := session.Loader{Storage: mem}
	if err := loader.Save(context.Background(), data); err != nil {
		return nil, errors.Wrap(err, "seed storage")
	}
	return mem, nil
}

// StorageFromInput — точка входа «воркера»: принимает Telethon-сессию (путь к
// SQLite `.session` файлу или StringSession-строку) и возвращает готовый
// session.Storage для gotd.
//
// Пример:
//
//	st, err := gotdbridge.StorageFromInput(sess)
//	if err != nil { ... }
//	client := telegram.NewClient(appID, appHash, telegram.Options{SessionStorage: st})
func StorageFromInput(input string) (*session.StorageMemory, error) {
	data, err := Convert(input)
	if err != nil {
		return nil, err
	}
	return Storage(data)
}
