// Command demo показывает интеграцию моста: берёт Telethon-сессию, подключается
// через gotd и печатает данные текущего аккаунта (Self).
//
// Использование:
//
//	APP_ID=... APP_HASH=... go run ./cmd/demo /path/to/account.session
//	APP_ID=... APP_HASH=... go run ./cmd/demo "1BQANOTEu...стро­ка_сессии"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/gotd/td/telegram"

	"github.com/n1s01/gotdbridge"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: demo <path-to.session | string-session>")
	}
	input := os.Args[1]

	appID, err := strconv.Atoi(os.Getenv("APP_ID"))
	if err != nil {
		log.Fatal("APP_ID env must be a valid integer")
	}
	appHash := os.Getenv("APP_HASH")
	if appHash == "" {
		log.Fatal("APP_HASH env is required")
	}

	// Мост: сессия Telethon → готовый session.Storage для gotd.
	st, err := gotdbridge.StorageFromInput(input)
	if err != nil {
		log.Fatalf("convert session: %v", err)
	}

	client := telegram.NewClient(appID, appHash, telegram.Options{
		SessionStorage: st,
	})

	ctx := context.Background()
	if err := client.Run(ctx, func(ctx context.Context) error {
		self, err := client.Self(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("authorized as: id=%d first=%q username=%q\n",
			self.ID, self.FirstName, self.Username)
		return nil
	}); err != nil {
		log.Fatalf("run: %v", err)
	}
}
