package whatsapp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/jamesread/japella/internal/utils"
	log "github.com/sirupsen/logrus"
)

type DeviceStore struct {
}

type WhatsAppConnector struct {
	utils.LogComponent

	deviceStore *DeviceStore
	clientLog   *WaLogAdaptor
}

func (c WhatsAppConnector) Start() {
	c.deviceStore = &DeviceStore{}
	c.clientLog = &WaLogAdaptor{Logger: log.StandardLogger()}

	c.connect()
}

func (conn WhatsAppConnector) connect() {
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:examplestore.db?_foreign_keys=on", conn.clientLog)
	if err != nil {
		panic(err)
	}

	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}

	client := whatsmeow.NewClient(deviceStore, conn.clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		connectWithQR(client)
	} else {
		mustConnect(client)
	}

	waitForInterruptThenDisconnect(client)
}

func connectWithQR(client *whatsmeow.Client) {
	qrChan, _ := client.GetQRChannel(context.Background())
	mustConnect(client)
	for evt := range qrChan {
		if evt.Event == "code" {
			// Render the QR code here
			// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
			fmt.Println("QR code:", evt.Code)
		} else {
			fmt.Println("Login event:", evt.Event)
		}
	}
}

func mustConnect(client *whatsmeow.Client) {
	if err := client.Connect(); err != nil {
		panic(err)
	}
}

func waitForInterruptThenDisconnect(client *whatsmeow.Client) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	client.Disconnect()
}

func eventHandler(evt any) {
	if v, ok := evt.(*events.Message); ok {
		fmt.Println("Received a message!", v.Message.GetConversation())
	}
}
