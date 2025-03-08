package dblogger

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	pb "github.com/jamesread/japella/gen/protobuf"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/botbase"
	"github.com/jamesread/japella/internal/runtimeconfig"
)

type DbLogger struct {
	botbase.Bot
}

func (bot DbLogger) Start() {
	bot.SetName("dblogger")
	bot.Logger().Infof("japella-bot-dblogger")

	cfg := runtimeconfig.Get().Database

	db := bot.ConnectDatabase(cfg)
	bot.ListenForMessages(db)
}

func (bot *DbLogger) Stop() {
}

func (bot *DbLogger) Name() string {
	return "dblogger"
}

func (bot *DbLogger) ListenForMessages(db *sql.DB) {
	handler := amqp.ConsumeForever("IncomingMessage", func(d amqp.Delivery) {
		msg := &pb.IncomingMessage{}

		amqp.Decode(d.Message.Body, &msg)

		bot.Logger().Infof("dblogger recv: %+v", msg)

		bot.handleMessage(db, msg)
	})

	handler.Wait()
	bot.Logger().Infof("done")
}

func (bot *DbLogger) ConnectDatabase(db *runtimeconfig.DatabaseConfig) *sql.DB {
	url := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", db.User, db.Password, db.Host, db.Database)

	conn, err := sql.Open("mysql", url)

	if err != nil {
		bot.Logger().Fatalf("Failed to connect to database: %v", err)
	}

	return conn
}

func (bot *DbLogger) handleMessage(db *sql.DB, msg *pb.IncomingMessage) {
	bot.Logger().Infof("handleMessage: %+v", msg)

	_, err := db.Exec("INSERT INTO messages (protocol, channel, author, content, timestamp) VALUES (?, ?, ?, ?, ?)", msg.Protocol, msg.Channel, msg.Author, msg.Content, msg.Timestamp)

	if err != nil {
		bot.Logger().Errorf("err: %v", err)
	}
}
