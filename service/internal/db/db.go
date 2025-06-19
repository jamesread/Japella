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
		&UserAccount{},
		&UserGroup{},
		&UserGroupMembership{},
		&ApiKey{},
		&Session{},
		&Cvar{},
	)

	if err != nil {
		log.Errorf("Error during database migration: %v", err)
		return
	}

	db.InsertCvarsIfNotExists()
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

func (db *DB) RegisterAccount(socialAccount *SocialAccount) (error) {
	res := db.conn.Create(socialAccount)

	return res.Error
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

func (db *DB) GetUserByApiKey(apiKey string) *ApiKey {
	ret := &ApiKey{}

	result := db.conn.Preload("UserAccount").Where("key_value = ?", apiKey).Limit(1).Find(ret)

	if result.Error != nil || result.RowsAffected == 0 {
		log.Warnf("No user found for API key: %s", apiKey)
		return nil
	}

	return ret
}

func (db *DB) GetUserByUsername(username string) *UserAccount {
	ret := &UserAccount{}

	result := db.conn.Where("username = ?", username).Limit(1).Find(ret)

	if result.Error != nil || ret.Username == "" {
		log.Warnf("No user found for username: %s", username)
		return nil
	}

	return ret
}

func (db *DB) CreateUserAccount(username, passwordHash string) (*UserAccount, error) {
	user := &UserAccount{
		Username:     username,
		PasswordHash: passwordHash,
	}

	result := db.conn.Create(user)

	if result.Error != nil {
		log.Errorf("Failed to create user account: %v", result.Error)
		return nil, result.Error
	}

	return user, nil
}

func (db *DB) CreateApiKey(user *UserAccount, keyValue string) (*ApiKey, error) {
	apiKey := &ApiKey{
		KeyValue:      keyValue,
		UserAccountID: user.ID,
		UserAccount:   *user,
	}

	result := db.conn.Create(apiKey)

	if result.Error != nil {
		log.Errorf("Failed to create API key: %v", result.Error)
		return nil, result.Error
	}

	return apiKey, nil
}

func (db *DB) HasAnyUsers() bool {
	var count int64
	result := db.conn.Model(&UserAccount{}).Count(&count)

	if result.Error != nil {
		log.Errorf("Failed to count users: %v", result.Error)
		return false
	}

	return count > 0
}

func (db *DB) CreateSession(sessionID string, uid uint32) error {
	session := &Session{
		UserAccountID: uid,
		SID:     sessionID,
	}

	result := db.conn.Create(session)

	if result.Error != nil {
		log.Errorf("Failed to create session: %v", result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) GetSessionByID(sessionID string) *Session {
	session := &Session{}

	result := db.conn.Preload("UserAccount").Where("s_id = ?", sessionID).First(session)

	if result.Error != nil {
		log.Warnf("No session found for ID: %s", sessionID)
		return nil
	}

	return session
}

func (db *DB) SelectUsers() ([]*UserAccount, error) {
	ret := make([]*UserAccount, 0)

	result := db.conn.Find(&ret)

	if result.Error != nil {
		log.Errorf("Failed to select users: %v", result.Error)
		return nil, result.Error
	}

	return ret, nil
}

func (db *DB) SelectAPIKeys() ([]*ApiKey, error) {
	ret := make([]*ApiKey, 0)

	result := db.conn.Preload("UserAccount").Find(&ret)

	if result.Error != nil {
		log.Errorf("Failed to select API keys: %v", result.Error)
		return nil, result.Error
	}

	return ret, nil
}

func (db *DB) SelectCvars() ([]*Cvar, error) {
	ret := make([]*Cvar, 0)

	result := db.conn.Find(&ret)

	if result.Error != nil {
		log.Errorf("Failed to select cvars: %v", result.Error)
		return nil, result.Error
	}

	return ret, nil
}

func (db *DB) GetCvarString(key string) (string) {
	var cvar Cvar

	result := db.conn.Where("key_name = ?", key).First(&cvar)

	if result.Error != nil {
		log.Errorf("Failed to get cvar %s: %v", key, result.Error)
		return ""
	}

	return cvar.ValueString
}

func (db *DB) GetCvarBool(key string) (bool) {
	var cvar Cvar

	result := db.conn.Where("key_name = ?", key).First(&cvar)

	if result.Error != nil {
		log.Errorf("Failed to get cvar %s: %v", key, result.Error)
		return false
	}

	if cvar.ValueInt == 1 {
		return true
	}

	return false;
}
