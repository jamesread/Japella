package dblogger

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/runtimeconfig"
	pb "github.com/jamesread/japella/gen/protobuf"
	"github.com/jamesread/japella/internal/botbase"
)

type DatabaseConfig struct {
	Host     string
	User     string
	Password string
	Database string
}

type LocalConfig struct {
	Common  *runtimeconfig.CommonConfig
	Database *DatabaseConfig
}

type DbLogger struct {
	botbase.Bot
}

func (bot *DbLogger) Start() {
	bot.SetName("dblogger")
	bot.Logger().Infof("japella-bot-dblogger")

	cfg := &LocalConfig{}
	cfg.Common = runtimeconfig.LoadNewConfigCommon()
	cfg.Database = &DatabaseConfig{}

	runtimeconfig.LoadConfig("config.database.yaml", cfg.Database)

	bot.Logger().Infof("cfg: %+v", cfg)

	db := bot.ConnectDatabase(cfg)
	bot.ListenForMessages(db)
}

func (bot *DbLogger) Stop() {
}

func (bot *DbLogger) Name() string {
	return "dblogger"
}

func (bot *DbLogger) ListenForMessages(db *sql.DB) {
	_, handler := amqp.ConsumeForever("IncommingMessage", func(d amqp.Delivery) {
		msg := &pb.IncommingMessage{}

		amqp.Decode(d.Message.Body, &msg)

		bot.Logger().Infof("dblogger recv: %+v", msg)

		bot.handleMessage(db, msg)
	})

	handler.Wait()
	bot.Logger().Infof("done")
}

func (bot *DbLogger) ConnectDatabase(cfg *LocalConfig) *sql.DB {
	url := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Database)

	db, err := sql.Open("mysql", url)

	if err != nil {
		bot.Logger().Fatalf("Failed to connect to database: %v", err)
	}

	return db
}

func (bot *DbLogger) handleMessage(db *sql.DB, msg *pb.IncommingMessage) {
	bot.Logger().Infof("handleMessage: %+v", msg)

	_, err := db.Exec("INSERT INTO messages (protocol, channel, author, content, timestamp) VALUES (?, ?, ?, ?, ?)", msg.Protocol, msg.Channel, msg.Author, msg.Content, msg.Timestamp)

	if err != nil {
		bot.Logger().Errorf("err: %v", err)
	}
}
