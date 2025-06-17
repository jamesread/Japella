package db

import (
	"gorm.io/gorm"
	"gorm.io/driver/mysql"

	"fmt"
	"github.com/jamesread/japella/internal/runtimeconfig"

	log "github.com/sirupsen/logrus"
)

type DB struct {
	conn *gorm.DB
}

func (db *DB) ReconnectDatabase(dbconfig runtimeconfig.DatabaseConfig) {
	if db.conn != nil {
		log.Warn("Database connection already exists, skipping reconnection")
		return
	}

	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", dbconfig.User, dbconfig.Password, dbconfig.Host, dbconfig.Database)

	var err error

	db.conn, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Warnf("Failed to connect to database: %v", err)
		return
	}

	db.Migrate()
}

func (db *DB) Migrate() {
	if db.conn == nil {
		log.Warn("Database connection is not established, cannot perform migration")
		return
	}

	err := db.conn.AutoMigrate(
		&SocialAccount{},
		&CannedPost{},
		&Post{},
	)

	if err != nil {
		log.Errorf("Error during database migration: %v", err)
		return
	}
}

func (db *DB) UpdateSocialAccountIdentity(id uint32, identity string) error {
	db.conn.Model(&SocialAccount{}).Where("id = ?", id).Update("identity", identity)

	return nil
}

func (db *DB) SelectSocialAccounts(onlyActive bool) []*SocialAccount {
	ret := make([]*SocialAccount, 0)

	if onlyActive {
		log.Infof("Selecting only active social accounts")
		db.conn.Where("active = ?", 1).Find(&ret)
	} else {
		db.conn.Find(&ret)
	}

	return ret
}

func (db *DB) SelectCannedPosts() []*CannedPost {
	ret := make([]*CannedPost, 0)

	db.conn.Find(&ret)

	return ret
}

func (db *DB) CreateCannedPost(content string) error {
	post := &CannedPost{
		Content: content,
	}

	db.conn.Create(post)

	return nil
}

func (db *DB) DeleteCannedPost(id uint32) error {
	log.Infof("Deleting canned post with ID: %s", id)

	db.conn.Delete(&CannedPost{}, id)

	return nil
}

func (db *DB) RegisterAccount(connector string, oauthToken string) error {
	db.conn.Create(&SocialAccount{
		Connector:  connector,
		Identity:   "?",
		OAuthToken: oauthToken,
	})

	return nil
}

func (db *DB) DeleteSocialAccount(id uint32) error {
	res := db.conn.Delete(&SocialAccount{}, id)

	return res.Error
}

func (db *DB) CreatePost(post *Post) error {
	db.conn.Create(post)

	return nil
}

func (db *DB) SelectPosts() ([]*Post, error) {
	ret := make([]*Post, 0)

	db.conn.Preload("SocialAccount").Order("id DESC").Find(&ret)

	return ret, nil
}

func (db *DB) GetSocialAccount(id uint32) (*SocialAccount, error) {
	var account SocialAccount

	result := db.conn.First(&account, id)

	if result.Error != nil {
		return nil, result.Error
	}

	return &account, nil
}

func (db *DB) SetSocialAccountActive(id uint32, active bool) error {
	result := db.conn.Model(&SocialAccount{}).Where("id = ?", id).Update("active", active)

	if result.Error != nil {
		return result.Error
	}

	return nil
}
