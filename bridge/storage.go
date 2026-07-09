package bridge

import (
	"context"

	"github.com/go-faster/errors"
	"github.com/gotd/td/session"
)

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

func StorageFromInput(input string) (*session.StorageMemory, error) {
	data, err := Convert(input)
	if err != nil {
		return nil, err
	}
	return Storage(data)
}
