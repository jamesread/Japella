package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"github.com/jamesread/japella/internal/amqp"
	log "github.com/sirupsen/logrus"
	"github.com/jamesread/japella/internal/runtimeconfig"
	pb "github.com/jamesread/japella/gen/protobuf"
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

func main() {
	log.Infof("japella-bot-dblogger")

	cfg := &LocalConfig{}
	cfg.Common = runtimeconfig.LoadNewConfigCommon()
	cfg.Database = &DatabaseConfig{}

	runtimeconfig.LoadConfig("config.database.yaml", cfg.Database)

	log.Infof("cfg: %+v", cfg)

	db := ConnectDatabase(cfg)
	ListenForMessages(db)
}

func ListenForMessages(db *sql.DB) {
	_, handler := amqp.ConsumeForever("IncommingMessage", func(d amqp.Delivery) {
		msg := &pb.IncommingMessage{}

		amqp.Decode(d.Message.Body, &msg)

		log.Infof("recv: %+v", msg)

		handleMessage(db, msg)
	})

	handler.Wait()
	log.Infof("done")
}

func ConnectDatabase(cfg *LocalConfig) *sql.DB {
	url := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Database)

	db, err := sql.Open("mysql", url)

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	return db
}

func handleMessage(db *sql.DB, msg *pb.IncommingMessage) {
	log.Infof("handleMessage: %+v", msg)

	_, err := db.Exec("INSERT INTO messages (channel, author, content) VALUES (?, ?, ?)", msg.Channel, msg.Author, msg.Content)

	if err != nil {
		log.Errorf("err: %v", err)
	}
}

