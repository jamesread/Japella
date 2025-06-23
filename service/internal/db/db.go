package db

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"fmt"

	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"

	log "github.com/sirupsen/logrus"
)

type DB struct {
	conn *gorm.DB
}

func (db *DB) ReconnectDatabase(dbconfig runtimeconfig.DatabaseConfig) error {
	if db.conn != nil {
		log.Warn("Database connection already exists, skipping reconnection")
		return nil
	}

	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v?charset=utf8&parseTime=True", dbconfig.User, dbconfig.Pass, dbconfig.Host, dbconfig.Name)

	var err error

	db.conn, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Warnf("Failed to connect to database: %v", err)
		return err
	}

	db.conn = db.conn.Set("gorm:table_options", "ENGINE=InnoDB CHARSET=utf8 COLLATE=utf8_bin")

	err = db.Migrate()

	if err != nil {
		log.Errorf("Failed to perform database migration: %v", err)
		return err
	}

	err = db.initAdminUser()

	if err != nil {
		log.Errorf("Failed to initialize admin user: %v", err)
		return err
	}

	db.conn = db.conn.Session(&gorm.Session{})

	return nil
}

func (db *DB) initAdminUser() error {
	if !db.HasAnyUsers() {
		log.Warn("No users found in the database, creating default admin user")

		passwordHash, err := utils.HashPassword("admin")

		if err != nil {
			log.Errorf("Error hashing default password: %v", err)
			return err
		}

		_, err = db.CreateUserAccount("admin", passwordHash)
		if err != nil {
			log.Errorf("Failed to create default admin user: %v", err)
			return err
		}
	}

	return nil
}

func (db *DB) Migrate() error {
	if db.conn == nil {
		log.Warn("Database connection is not established, cannot perform migration")
		return fmt.Errorf("database connection is not established")
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
		return err
	}

	err = db.InsertCvarsIfNotExists()

	return err
}

func (db *DB) UpdateSocialAccountIdentity(id uint32, identity string) error {
	result := db.conn.Model(&SocialAccount{}).Where("id = ?", id).Update("identity", identity)

	if result.Error != nil {
		log.Errorf("Failed to update social account identity: %v", result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) SelectSocialAccounts(onlyActive bool) []*SocialAccount {
	ret := make([]*SocialAccount, 0)

	if onlyActive {
		log.Infof("Selecting only active social accounts")
		result := db.conn.Where("active = ?", 1).Find(&ret)
		if result.Error != nil {
			log.Errorf("Failed to select active social accounts: %v", result.Error)
			return ret
		}
	} else {
		result := db.conn.Find(&ret)
		if result.Error != nil {
			log.Errorf("Failed to select social accounts: %v", result.Error)
			return ret
		}
	}

	return ret
}

func (db *DB) SelectCannedPosts() []*CannedPost {
	ret := make([]*CannedPost, 0)

	result := db.conn.Find(&ret)

	if result.Error != nil {
		log.Errorf("Failed to select canned posts: %v", result.Error)
		return ret
	}

	return ret
}

func (db *DB) CreateCannedPost(content string) error {
	post := &CannedPost{
		Content: content,
	}

	result := db.conn.Create(post)

	if result.Error != nil {
		log.Errorf("Failed to create canned post: %v", result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) DeleteCannedPost(id uint32) error {
	log.Infof("Deleting canned post with ID: %d", id)

	result := db.conn.Delete(&CannedPost{}, id)

	if result.Error != nil {
		log.Errorf("Failed to delete canned post: %v", result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) RegisterAccount(socialAccount *SocialAccount) error {
	result := db.conn.Create(socialAccount)

	if result.Error != nil {
		log.Errorf("Failed to register social account: %v", result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) DeleteSocialAccount(id uint32) error {
	result := db.conn.Delete(&SocialAccount{}, id)

	if result.Error != nil {
		log.Errorf("Failed to delete social account: %v", result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) CreatePost(post *Post) error {
	result := db.conn.Create(post)

	if result.Error != nil {
		log.Errorf("Failed to create post: %v", result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) SelectPosts() ([]*Post, error) {
	ret := make([]*Post, 0)

	result := db.conn.Preload("SocialAccount").Order("id DESC").Find(&ret)

	if result.Error != nil {
		log.Errorf("Failed to select posts: %v", result.Error)
		return nil, result.Error
	}

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
		log.Errorf("Failed to set social account active status: %v", result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) GetUserByApiKey(apiKey string) *UserAccount {
	ret := &ApiKey{}

	result := db.conn.Preload("UserAccount").Where("key_value = ?", apiKey).First(ret)

	if result.Error != nil || result.RowsAffected == 0 {
		return nil
	}

	return ret.UserAccount
}

func (db *DB) GetUserByUsername(username string) *UserAccount {
	var ret UserAccount

	result := db.conn.Where("username = ?", username).First(&ret)

	if result.Error != nil || ret.Username == "" {
		log.Warnf("No user found for username: %s", username)
		return nil
	}

	return &ret
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
		UserAccount:   user,
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
		SID:           sessionID,
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

func (db *DB) GetCvar(key string) *Cvar {
	var cvar Cvar

	result := db.conn.Where("key_name = ?", key).First(&cvar)

	if result.Error != nil || result.RowsAffected == 0 {
		return nil
	}

	return &cvar
}

func (db *DB) GetCvarString(key string) string {
	var cvar Cvar

	result := db.conn.Where("key_name = ?", key).First(&cvar)

	if result.Error != nil {
		log.Errorf("Failed to get cvar %s: %v", key, result.Error)
		return ""
	}

	return cvar.ValueString
}

func (db *DB) GetCvarBool(key string) bool {
	var cvar Cvar

	result := db.conn.Where("key_name = ?", key).First(&cvar)

	if result.Error != nil {
		log.Errorf("Failed to get cvar %s: %v", key, result.Error)
		return false
	}

	if cvar.ValueInt == 1 {
		return true
	}

	return false
}

func (db *DB) GetCvarInt(key string) int32 {
	var cvar Cvar

	result := db.conn.Where("key_name = ?", key).First(&cvar)

	if result.Error != nil {
		log.Errorf("Failed to get cvar %s: %v", key, result.Error)
		return 0
	}

	return cvar.ValueInt
}

func (db *DB) SetCvarString(key, value string) error {
	result := db.conn.Model(&Cvar{}).Where("key_name = ?", key).Update("value_string", value)

	if result.Error != nil {
		log.Errorf("Failed to set cvar %s: %v", key, result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) SetCvarBool(key string, value bool) error {
	var intValue int32

	if value {
		intValue = 1
	} else {
		intValue = 0
	}

	return db.SetCvarInt(key, intValue)
}

func (db *DB) SetCvarInt(key string, value int32) error {
	result := db.conn.Model(&Cvar{}).Where("key_name = ?", key).Update("value_int", value)

	if result.Error != nil {
		log.Errorf("Failed to set cvar %s: %v", key, result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) SaveUserPreferences(preferences *UserPreferences) error {
	result := db.conn.Save(preferences)

	if result.Error != nil {
		log.Errorf("Failed to save user preferences: %v", result.Error)
		return result.Error
	}

	return nil
}

func (db *DB) RevokeApiKey(id uint32) error {
	result := db.conn.Delete(&ApiKey{}, id)

	if result.Error != nil {
		log.Errorf("Failed to revoke API key: %v", result.Error)
		return result.Error
	}

	return nil
}
