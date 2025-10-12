package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jamesread/golure/pkg/dirs"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	"github.com/jmoiron/sqlx"

	"sync"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type DB struct {
	utils.LogComponent

	dsn           string
	migrationsDir string

	errorMessage string

	dbconfig runtimeconfig.DatabaseConfig

	connectionMutex sync.Mutex

	connx                *sqlx.DB
	currentSchemaVersion uint
	currentSchemaDirty   bool
}

func (db *DB) SetErrorMessage(message string) {
	db.errorMessage = message
}

func (db *DB) GetErrorMessage() string {
	return db.errorMessage
}

func (db *DB) SetDatabaseConfig(dbconfig runtimeconfig.DatabaseConfig) {
	db.SetPrefix("db")
	db.dbconfig = dbconfig
}

func (db *DB) GetDatabaseHost() string {
	return db.dbconfig.Host
}

func (db *DB) GetDatabaseName() string {
	return db.dbconfig.Name
}

func (db *DB) ReconnectLoop() {
	for {
		db.ReconnectDatabaseAndSetErrorMessage()

		time.Sleep(30 * time.Second)
	}
}

func (db *DB) ReconnectDatabaseAndSetErrorMessage() {
	if err := db.reconnectDatabase(); err != nil {
		db.SetErrorMessage(err.Error())
	} else {
		db.SetErrorMessage("")
	}
}

func (db *DB) findMigrationsDirectory() (string, error) {
	toSearch := []string{
		"../var/app-skel/db-migrations/",
		"../../../var/app-skel/db-migrations/", // Relative to this file, for unit tests
		"/app/db-migrations/",
	}

	return dirs.GetFirstExistingDirectory("db-migrations", toSearch)
}

func (db *DB) Migrate(chain *ConnectionChain) {
	db.Logger().Infof("Starting migration from directory: %s", db.migrationsDir)

	m, err := migrate.New(
		"file://"+db.migrationsDir,
		"mysql://"+db.dsn,
	)

	if err != nil {
		db.Logger().Errorf("Failed to create migration instance: %v", err)
		chain.err = err
		return
	}

	db.currentSchemaVersion, db.currentSchemaDirty, _ = m.Version()
	db.Logger().Infof("Current schema version: %v, dirty: %v", db.currentSchemaVersion, db.currentSchemaDirty)

	err = m.Up()

	if err == migrate.ErrNoChange {
		db.Logger().Info("Database is already at the latest version, no migration needed")
	} else if err != nil {
		db.Logger().Errorf("Failed to migrate database: %v", err)
		chain.err = err
	} else {
		db.currentSchemaVersion, db.currentSchemaDirty, _ = m.Version()
		db.Logger().Infof("Database upgraded to schema version: %v, dirty: %v", db.currentSchemaVersion, db.currentSchemaDirty)
	}
}

type databaseConnectionCheck func(*ConnectionChain)

func (db *DB) checkConnNotNil(chain *ConnectionChain) {
	if db.connx != nil {
		if db.connx.Ping() == nil {
			db.Logger().Debugf("Database connection is alive, skipping reconnection")
			chain.continueConnecting = false
		} else {
			db.Logger().Warn("Database connection is not alive, reconnecting")

			db.connx.Close()
			db.connx = nil
		}
	}
}

func (db *DB) buildDsn(chain *ConnectionChain) {
	db.dsn = fmt.Sprintf("%v:%v@tcp(%v)/%v?charset=utf8&parseTime=True", db.dbconfig.User, db.dbconfig.Pass, db.dbconfig.Host, db.dbconfig.Name)
}

func (db *DB) connectToDatabase(chain *ConnectionChain) {
	if db.connx == nil {
		var err error

		db.connx, err = sqlx.Connect("mysql", db.dsn)

		if err != nil {
			chain.err = errors.New("failed to connect to database: " + err.Error())
		}
	}
}

func (db *DB) pingOk(chain *ConnectionChain) {
	chain.err = db.connx.Ping()
}

func (db *DB) findMigrationsDirectoryWrapped(chain *ConnectionChain) {
	md, err := db.findMigrationsDirectory()

	if err != nil {
		db.Logger().Errorf("Failed to find migrations directory: %v", err)
		chain.err = err
		return
	}

	db.migrationsDir = md
}

type ConnectionChain struct {
	err                error
	continueConnecting bool
}

func (db *DB) reconnectDatabase() error {
	db.connectionMutex.Lock()
	defer db.connectionMutex.Unlock()

	db.Logger().Debugf("Reconnecting to database")

	chain := &ConnectionChain{
		continueConnecting: true,
	}

	var connectionChecks = []databaseConnectionCheck{
		db.checkConnNotNil,
		db.buildDsn,
		db.connectToDatabase,
		db.pingOk,
		db.findMigrationsDirectoryWrapped,
		db.Migrate,
		db.InsertCvarsIfNotExists,
		db.initAdminUser,
	}

	for _, check := range connectionChecks {
		check(chain)

		if chain.err != nil {
			db.Logger().Errorf("Database connection check failed: %v", chain.err)
			return chain.err
		}

		if !chain.continueConnecting {
			return nil
		}
	}

	return nil
}

func (db *DB) initAdminUser(chain *ConnectionChain) {
	if !db.HasAnyUsers() {
		db.Logger().Warn("No users found in the database, creating default admin user")

		passwordHash, err := utils.HashPassword("admin")

		if err != nil {
			db.Logger().Errorf("Error hashing default password: %v", err)
			chain.err = err
			return
		}

		_, err = db.CreateUserAccount("admin", passwordHash)
		if err != nil {
			db.Logger().Errorf("Failed to create default admin user: %v", err)
			chain.err = err
			return
		}
	}
}

func (db *DB) ResilientExec(query string, args ...any) (sql.Result, error) {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()

		return nil, errors.New("database connection is not established")
	}

	return db.connx.Exec(query, args...)
}

func (db *DB) ResilientNamedExec(query string, args ...any) (sql.Result, error) {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()

		return nil, errors.New("database connection is not established")
	}

	return db.connx.NamedExec(query, args)
}

func (db *DB) ResilientSelect(dest interface{}, query string, args ...any) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()

		return errors.New("database connection is not established")
	}

	return db.connx.Select(dest, query, args...)
}

func (db *DB) ResilientGet(dest interface{}, query string, args ...any) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()

		return errors.New("database connection is not established")
	}

	return db.connx.Get(dest, query, args...)
}

func (db *DB) UpdateSocialAccountIdentity(id uint32, identity string) error {
	_, err := db.ResilientExec("UPDATE social_accounts SET identity = ?, updated_at = NOW() WHERE id = ?", identity, id)
	if err != nil {
		db.Logger().Errorf("Failed to update social account identity: %v", err)
		return err
	}
	return nil
}

func (db *DB) SelectSocialAccounts(onlyActive bool) []*SocialAccount {
	ret := make([]*SocialAccount, 0)
	var err error
	if onlyActive {
		db.Logger().Infof("Selecting only active social accounts")
		err = db.ResilientSelect(&ret, "SELECT * FROM social_accounts WHERE active = 1")
	} else {
		err = db.ResilientSelect(&ret, "SELECT * FROM social_accounts")
	}
	if err != nil {
		db.Logger().Errorf("Failed to select social accounts: %v", err)
	}
	return ret
}

func (db *DB) SelectCannedPosts() []*CannedPost {
	ret := make([]*CannedPost, 0)
	err := db.ResilientSelect(&ret, "SELECT * FROM canned_posts")
	if err != nil {
		db.Logger().Errorf("Failed to select canned posts: %v", err)
	}
	return ret
}

func (db *DB) GetCannedPost(id uint32) (*CannedPost, error) {
	var cannedPost CannedPost
	err := db.ResilientGet(&cannedPost, "SELECT * FROM canned_posts WHERE id = ? LIMIT 1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			db.Logger().Warnf("No canned post found for ID: %d", id)
		} else {
			db.Logger().Errorf("Error querying canned post by ID: %v", err)
		}
		return nil, err
	}
	return &cannedPost, nil
}

func (db *DB) CreateCannedPost(content string) error {
	_, err := db.ResilientExec("INSERT INTO canned_posts (content, created_at, updated_at) VALUES (?, NOW(), NOW())", content)
	if err != nil {
		db.Logger().Errorf("Failed to create canned post: %v", err)
		return err
	}
	return nil
}

func (db *DB) DeleteCannedPost(id uint32) error {
	db.Logger().Infof("Deleting canned post with ID: %d", id)
	_, err := db.ResilientExec("DELETE FROM canned_posts WHERE id = ?", id)
	if err != nil {
		db.Logger().Errorf("Failed to delete canned post: %v", err)
		return err
	}
	return nil
}

func (db *DB) RegisterAccount(socialAccount *SocialAccount) error {
	_, err := db.ResilientNamedExec(`INSERT INTO social_accounts (connector, identity, oauth2_token, oauth2_token_expiry, oauth2_refresh_token, active, dpop_key, created_at, updated_at) VALUES (:connector, :identity, :oauth2_token, :oauth2_token_expiry, :oauth2_refresh_token, :active, :dpop_key, NOW(), NOW())`, socialAccount)
	if err != nil {
		db.Logger().Errorf("Failed to register social account: %v", err)
		return err
	}
	return nil
}

func (db *DB) DeleteSocialAccount(id uint32) error {
	_, err := db.ResilientExec("DELETE FROM social_accounts WHERE id = ?", id)
	if err != nil {
		db.Logger().Errorf("Failed to delete social account: %v", err)
		return err
	}
	return nil
}

func (db *DB) CreatePost(post *Post) error {
	_, err := db.ResilientNamedExec(`INSERT INTO posts (social_account_id, status, content, post_url, remote_id, created_at, updated_at) VALUES (:social_account_id, :status, :content, :post_url, :remote_id, NOW(), NOW())`, post)
	if err != nil {
		db.Logger().Errorf("Failed to create post: %v", err)
		return err
	}
	return nil
}

func (db *DB) SelectPosts() ([]*Post, error) {
	ret := make([]*Post, 0)
	err := db.ResilientSelect(&ret, "SELECT p.id, p.social_account_id, p.status, p.content, p.post_url, p.remote_id, p.created_at, p.campaign_id AS campaign_id, c.name AS campaign_name FROM posts p LEFT JOIN campaigns c ON p.campaign_id = c.id ORDER BY p.id DESC")
	if err != nil {
		db.Logger().Errorf("Failed to select posts: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) GetSocialAccount(id uint32) (*SocialAccount, error) {
	var account SocialAccount
	err := db.ResilientGet(&account, "SELECT * FROM social_accounts WHERE id = ?", id)
	if err != nil {
		db.Logger().Errorf("Failed to get social account: %v", err)
		return nil, err
	}
	return &account, nil
}

func (db *DB) SetSocialAccountActive(id uint32, active bool) error {
	_, err := db.ResilientExec("UPDATE social_accounts SET active = ?, updated_at = NOW() WHERE id = ?", active, id)
	if err != nil {
		db.Logger().Errorf("Failed to set social account active status: %v", err)
		return err
	}
	return nil
}

func (db *DB) GetUserByApiKey(apiKey string) *UserAccount {
	var user UserAccount
	err := db.ResilientGet(&user, `SELECT ua.* FROM user_accounts ua JOIN api_keys ak ON ua.id = ak.user_account_id WHERE ak.key_value = ? LIMIT 1`, apiKey)
	if err != nil {
		if err == sql.ErrNoRows {
			db.Logger().Errorf("Failed to get user by API key: %v", err)
		}
		return nil
	}
	return &user
}

func (db *DB) GetUserByUsername(username string) *UserAccount {
	var user UserAccount
	err := db.ResilientGet(&user, "SELECT * FROM user_accounts WHERE username = ? LIMIT 1", username)
	if err != nil {
		if err == sql.ErrNoRows {
			db.Logger().Warnf("No user found for username: %s", username)
		} else {
			db.Logger().Errorf("Error querying user by username: %v", err)
		}
		return nil
	}
	return &user
}

func (db *DB) CreateUserAccount(username, passwordHash string) (*UserAccount, error) {
	res, err := db.ResilientExec("INSERT INTO user_accounts (username, password_hash, created_at, updated_at) VALUES (?, ?, NOW(), NOW())", username, passwordHash)
	if err != nil {
		db.Logger().Errorf("Failed to create user account: %v", err)
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &UserAccount{
		Model:        Model{ID: uint32(id)},
		Username:     username,
		PasswordHash: passwordHash,
	}, nil
}

func (db *DB) CreateApiKey(user *UserAccount, keyValue string) (*ApiKey, error) {
	res, err := db.ResilientExec("INSERT INTO api_keys (key_value, user_account_id, created_at, updated_at) VALUES (?, ?, NOW(), NOW())", keyValue, user.ID)
	if err != nil {
		db.Logger().Errorf("Failed to create API key: %v", err)
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &ApiKey{
		Model:         Model{ID: uint32(id)},
		KeyValue:      keyValue,
		UserAccountID: user.ID,
		UserAccount:   user,
	}, nil
}

func (db *DB) HasAnyUsers() bool {
	var count int64
	err := db.ResilientGet(&count, "SELECT COUNT(*) FROM user_accounts")
	if err != nil {
		db.Logger().Errorf("Failed to count users: %v", err)
		return false
	}
	return count > 0
}

func (db *DB) CreateSession(sessionID string, uid uint32) error {
	_, err := db.ResilientExec("INSERT INTO sessions (user_account_id, sid, created_at, updated_at) VALUES (?, ?, NOW(), NOW())", uid, sessionID)
	if err != nil {
		db.Logger().Errorf("Failed to create session: %v", err)
		return err
	}
	return nil
}

func (db *DB) GetUserByID(id uint32) *UserAccount {
	var user UserAccount
	err := db.ResilientGet(&user, "SELECT * FROM user_accounts WHERE id = ? LIMIT 1", id)
	if err != nil {
		db.Logger().Errorf("Failed to get user by ID: %v", err)
		return nil
	}
	return &user
}

func (db *DB) GetSessionByID(sessionID string) *Session {
	var session Session
	err := db.ResilientGet(&session, "SELECT * FROM sessions WHERE sid = ? LIMIT 1", sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			db.Logger().Warnf("No session found for ID: %s", sessionID)
		} else {
			db.Logger().Errorf("Error querying session by ID: %v", err)
		}
		return nil
	}

	session.UserAccount = db.GetUserByID(session.UserAccountID)
	if session.UserAccount == nil {
		db.Logger().Warnf("No user found for session ID: %s", sessionID)
		return nil
	}

	return &session
}

func (db *DB) SelectUsers() ([]*UserAccount, error) {
	ret := make([]*UserAccount, 0)
	err := db.ResilientSelect(&ret, "SELECT * FROM user_accounts")
	if err != nil {
		db.Logger().Errorf("Failed to select users: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) SelectAPIKeys() ([]*ApiKey, error) {
	ret := make([]*ApiKey, 0)
	err := db.ResilientSelect(&ret, "SELECT * FROM api_keys")
	if err != nil {
		db.Logger().Errorf("Failed to select API keys: %v", err)
		return nil, err
	}

	db.Logger().Infof("Selected %+v API keys", ret)
	return ret, nil
}

func (db *DB) SelectCvars() ([]*Cvar, error) {
	ret := make([]*Cvar, 0)
	err := db.ResilientSelect(&ret, "SELECT * FROM cvars")
	if err != nil {
		db.Logger().Errorf("Failed to select cvars: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) GetCvar(key string) *Cvar {
	var cvar Cvar
	err := db.ResilientGet(&cvar, "SELECT * FROM cvars WHERE key_name = ? LIMIT 1", key)
	if err != nil {
		if err == sql.ErrNoRows {
			db.Logger().Warnf("No cvar found for key: %s", key)
		} else {
			db.Logger().Errorf("Error querying cvar by key: %v", err)
		}
		return nil
	}
	return &cvar
}

func (db *DB) GetCvarString(key string) string {
	var cvar Cvar
	err := db.ResilientGet(&cvar, "SELECT * FROM cvars WHERE key_name = ? LIMIT 1", key)
	if err != nil {
		db.Logger().Errorf("Failed to get cvar %s: %v", key, err)
		return ""
	}
	return cvar.ValueString
}

func (db *DB) GetCvarBool(key string) bool {
	var cvar Cvar
	err := db.ResilientGet(&cvar, "SELECT * FROM cvars WHERE key_name = ? LIMIT 1", key)
	if err != nil {
		db.Logger().Errorf("Failed to get cvar %s: %v", key, err)
		return false
	}
	return cvar.ValueInt == 1
}

func (db *DB) GetCvarInt(key string) int32 {
	var cvar Cvar
	err := db.ResilientGet(&cvar, "SELECT * FROM cvars WHERE key_name = ? LIMIT 1", key)
	if err != nil {
		db.Logger().Errorf("Failed to get cvar %s: %v", key, err)
		return 0
	}
	return cvar.ValueInt
}

func (db *DB) SetCvarString(key, value string) error {
	_, err := db.ResilientExec("UPDATE cvars SET value_string = ?, updated_at = NOW() WHERE key_name = ?", value, key)
	if err != nil {
		db.Logger().Errorf("Failed to set cvar %s: %v", key, err)
		return err
	}
	return nil
}

func (db *DB) SetCvarBool(key string, value bool) error {
	intValue := int32(0)
	if value {
		intValue = 1
	}
	return db.SetCvarInt(key, intValue)
}

func (db *DB) SetCvarInt(key string, value int32) error {
	db.Logger().Infof("Setting cvar %s to value %d", key, value)

	_, err := db.ResilientExec("UPDATE cvars SET value_int = ?, updated_at = NOW() WHERE key_name = ?", value, key)
	if err != nil {
		db.Logger().Errorf("Failed to set cvar %s: %v", key, err)
		return err
	}
	return nil
}

func (db *DB) SaveUserPreferences(preferences *UserPreferences) error {
	_, err := db.ResilientNamedExec(`INSERT INTO user_preferences (user_account_id, language, created_at, updated_at) VALUES (:user_account_id, :language, NOW(), NOW()) ON DUPLICATE KEY UPDATE language = VALUES(language), updated_at = NOW()`, preferences)
	if err != nil {
		db.Logger().Errorf("Failed to save user preferences: %v", err)
		return err
	}
	return nil
}

func (db *DB) RevokeApiKey(id uint32) error {
	_, err := db.ResilientExec("DELETE FROM api_keys WHERE id = ?", id)
	if err != nil {
		db.Logger().Errorf("Failed to revoke API key: %v", err)
		return err
	}
	return nil
}

func (db *DB) UpdateSocialAccountToken(socialAccountId uint32, accessToken string, refreshToken string, epiresIn int64) error {
	db.Logger().Infof("Updating social account token for ID: %d", socialAccountId)

	_, err := db.ResilientExec("UPDATE social_accounts SET oauth2_token = ?, oauth2_refresh_token = ?, oauth2_token_expiry = DATE_ADD(NOW(), INTERVAL ? SECOND), updated_at = NOW() WHERE id = ?", accessToken, refreshToken, epiresIn, socialAccountId)

	if err != nil {
		db.Logger().Errorf("Failed to update social account token: %v", err)
		return err
	}

	return nil
}

func (db *DB) CreateCampaign(campaign *Campaign) error {
	_, err := db.ResilientNamedExec(`INSERT INTO campaigns (name, description, start_date, end_date, created_at, updated_at) VALUES (:name, :description, :start_date, :end_date, NOW(), NOW())`, campaign)
	if err != nil {
		db.Logger().Errorf("Failed to create campaign: %v", err)
		return err
	}
	return nil
}

func (db *DB) SelectCampaigns() ([]*Campaign, error) {
	ret := make([]*Campaign, 0)
	err := db.ResilientSelect(&ret, "SELECT c.*, count(p.id) as post_count, max(p.created_at) AS last_post_date FROM campaigns c LEFT JOIN posts p ON p.campaign_id = c.id GROUP BY c.id ORDER BY id DESC")
	if err != nil {
		db.Logger().Errorf("Failed to select campaigns: %v", err)
		return nil, err
	}
	return ret, nil
}

func (db *DB) UpdateCampaign(campaign *Campaign) error {
	_, err := db.ResilientNamedExec(`UPDATE campaigns SET name = :name, description = :description, start_date = :start_date, end_date = :end_date, updated_at = NOW() WHERE id = :id`, campaign)
	if err != nil {
		db.Logger().Errorf("Failed to update campaign: %v", err)
		return err
	}
	return nil
}

func (db *DB) DeleteCampaign(id uint32) error {
	_, err := db.ResilientExec("DELETE FROM campaigns WHERE id = ?", id)
	if err != nil {
		db.Logger().Errorf("Failed to delete campaign: %v", err)
		return err
	}

	return nil
}

func (db *DB) GetCampaign(id uint32) (*Campaign, error) {
	var campaign Campaign
	err := db.ResilientGet(&campaign, "SELECT * FROM campaigns WHERE id = ? LIMIT 1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			db.Logger().Warnf("No campaign found for ID: %d", id)
		} else {
			db.Logger().Errorf("Error querying campaign by ID: %v", err)
		}
		return nil, err
	}
	return &campaign, nil
}

func (db *DB) UpdateCannedPost(cannedPost *CannedPost) error {
	_, err := db.ResilientNamedExec(`UPDATE canned_posts SET content = :content, updated_at = NOW() WHERE id = :id`, cannedPost)

	if err != nil {
		db.Logger().Errorf("Failed to update canned post: %v", err)
		return err
	}

	return nil
}
