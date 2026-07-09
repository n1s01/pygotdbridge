package bridge

import (
	"net"
	"strconv"

	"github.com/go-faster/errors"
)

const dcPort = 443

var prodDC = map[int]string{
	1: "149.154.175.53",
	2: "149.154.167.51",
	3: "149.154.175.100",
	4: "149.154.167.91",
	5: "91.108.56.130",
}

var testDC = map[int]string{
	1: "149.154.175.10",
	2: "149.154.167.40",
	3: "149.154.175.117",
}

func dcAddr(dcID int, test bool) (string, error) {
	table := prodDC
	if test {
		table = testDC
	}
	host, ok := table[dcID]
	if !ok {
		return "", errors.Errorf("unknown dc_id %d (test=%v)", dcID, test)
	}
	return net.JoinHostPort(host, strconv.Itoa(dcPort)), nil
}
