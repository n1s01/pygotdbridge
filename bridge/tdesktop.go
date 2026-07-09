package bridge

import (
	"github.com/go-faster/errors"
	"github.com/gotd/td/session"
	"github.com/gotd/td/session/tdesktop"
)

func FromTDesktopAll(root string, passcode []byte) ([]*session.Data, error) {
	accounts, err := tdesktop.Read(root, passcode)
	if err != nil {
		return nil, errors.Wrap(err, "read tdata")
	}
	if len(accounts) == 0 {
		return nil, errors.New("tdata has no accounts")
	}

	out := make([]*session.Data, 0, len(accounts))
	for i, acc := range accounts {
		data, err := session.TDesktopSession(acc)
		if err != nil {
			return nil, errors.Wrapf(err, "convert account %d", i)
		}
		out = append(out, data)
	}
	return out, nil
}

func FromTDesktop(root string, passcode []byte) (*session.Data, error) {
	all, err := FromTDesktopAll(root, passcode)
	if err != nil {
		return nil, err
	}
	return all[0], nil
}

func StorageFromTDesktop(root string, passcode []byte) (*session.StorageMemory, error) {
	data, err := FromTDesktop(root, passcode)
	if err != nil {
		return nil, err
	}
	return Storage(data)
}
