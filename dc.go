package gotdbridge

import (
	"net"
	"strconv"

	"github.com/go-faster/errors"
)

// В отличие от Telethon, Pyrogram не хранит адрес дата-центра в сессии — только
// его номер (dc_id). Адрес восстанавливаем по фиксированной таблице продакшн- и
// тестовых DC Telegram (порт всегда 443).
//
// См. https://core.telegram.org/api/datacenter — это те же IP, что Telethon
// записывает в свой .session.
const dcPort = 443

// prodDC — IPv4-адреса продакшн дата-центров по dc_id.
var prodDC = map[int]string{
	1: "149.154.175.53",
	2: "149.154.167.51",
	3: "149.154.175.100",
	4: "149.154.167.91",
	5: "91.108.56.130",
}

// testDC — IPv4-адреса тестовых дата-центров по dc_id.
var testDC = map[int]string{
	1: "149.154.175.10",
	2: "149.154.167.40",
	3: "149.154.175.117",
}

// dcAddr возвращает "host:443" для заданного dc_id. test выбирает таблицу
// тестовых DC (Pyrogram хранит флаг test_mode в сессии).
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
