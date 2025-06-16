package db

import (
	"database/sql"
	"database/sql/driver"

	"context"
	"errors"

	mysql "github.com/go-sql-driver/mysql"

	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jamesread/japella/internal/runtimeconfig"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"

	"github.com/shogo82148/go-sql-proxy"

	log "github.com/sirupsen/logrus"
)

var hooker *proxy.HooksContext

type DB struct {
	conn *sql.DB
}

func (db *DB) UpdateSocialAccountIdentity(id string, identity string) error {
	query := "UPDATE accounts SET identity = ? WHERE id = ?"
	_, err := db.conn.Exec(query, identity, id)

	if err != nil {
		log.Errorf("Error updating social account identity: %v", err)
		return fmt.Errorf("error updating social account identity: %w", err)
	}

	return err
}

func (db *DB) ReconnectDatabase(dbconfig runtimeconfig.DatabaseConfig) {
	hooker = &proxy.HooksContext{
		Open: func(_ context.Context, _ interface{}, conn *proxy.Conn) error {
			log.Println("SQL Open")
			return nil
		},
		Exec: func(_ context.Context, _ interface{}, stmt *proxy.Stmt, args []driver.NamedValue, result driver.Result) error {
			log.Printf("SQL Exec: %s; args = %+v\n", stmt.QueryString, args)
			return nil
		},
		Query: func(_ context.Context, _ interface{}, stmt *proxy.Stmt, args []driver.NamedValue, rows driver.Rows) error {
			log.Printf("SQL Query: %s; args = %+v\n", stmt.QueryString, args)
			return nil
		},
	}

	sql.Register("mysql-logged", proxy.NewProxyContext(
		&mysql.MySQLDriver{},
		hooker,
	))

	if !dbconfig.Enabled {
		log.Warnf("Database is not enabled in configuration, skipping connection")
		return
	}

	url := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", dbconfig.User, dbconfig.Password, dbconfig.Host, dbconfig.Database)

	var err error

	db.conn, err = sql.Open("mysql-logged", url)

	if err != nil {
		log.Warnf("Failed to connect to database: %v", err)
	}
}

type SocialAccount struct {
	Id         string
	Connector  string
	Identity   string
	OAuthToken string
}

func (db *DB) SelectSocialAccounts() map[string]*SocialAccount {
	socialaccounts := make(map[string]*SocialAccount)

	if db.conn == nil {
		log.Warnf("Database connection is not established, cannot load social accounts")
		return socialaccounts
	}

	sql := "SELECT id, connector, oauthToken, identity FROM accounts"
	rows, err := db.conn.Query(sql)

	if err != nil {
		log.Errorf("Error querying social accounts: %v", err)
		return socialaccounts
	}

	defer rows.Close()

	for rows.Next() {
		var id, connectorName, oauthToken, identity string

		if err := rows.Scan(&id, &connectorName, &oauthToken, &identity); err != nil {
			log.Errorf("Error scanning social account: %v", err)
			continue
		}

		socialaccounts[id] = &SocialAccount{
			Id:         id,
			Connector:  connectorName,
			Identity:   identity,
			OAuthToken: oauthToken,
		}
	}

	return socialaccounts
}

type CannedPost struct {
	Id      string
	Content string
}

func (db *DB) SelectCannedPosts() map[string]*CannedPost {
	cannedPosts := make(map[string]*CannedPost)

	if db.conn == nil {
		log.Warnf("Database connection is not established, cannot load canned posts")
		return cannedPosts
	}

	sql := "SELECT id, content FROM canned_posts"
	rows, err := db.conn.Query(sql)

	if err != nil {
		log.Errorf("Error querying canned posts: %v", err)
		return cannedPosts
	}

	defer rows.Close()

	for rows.Next() {
		var id, content string

		if err := rows.Scan(&id); err != nil {
			log.Errorf("Error scanning canned post: %v", err)
			continue
		}

		cannedPosts[id] = &CannedPost{
			Id: id,
			Content: content,
		}
	}

	return cannedPosts
}

func (db *DB) CreateCannedPost(content string) error {
	if db.conn == nil {
		return fmt.Errorf("database connection is not established")
	}

	sql := "INSERT INTO canned_posts (content) VALUES (?)"

	_, err := db.conn.Exec(sql, content)

	return err
}

func (db *DB) DeleteCannedPost(id string) error {
	log.Infof("Deleting canned post with ID: %s", id)

	sql := "DELETE FROM canned_posts WHERE id = ?"

	_, err := db.conn.Exec(sql, id)

	return err
}

func (db *DB) RegisterAccount(connector string, oauthToken string) error {
	if db.conn == nil {
		return errors.New("database connection is not established")
	}

	sql := "INSERT INTO accounts (connector, identity, oauthToken) VALUES (?, ?, ?)"
	_, err := db.conn.Exec(sql, connector, "?", oauthToken)

	if err != nil {
		return fmt.Errorf("error inserting social account: %v", err)
	}

	return nil
}

func (db *DB) DeleteSocialAccount(id string) error {
	if db.conn == nil {
		return fmt.Errorf("database connection is not established")
	}

	sql := "DELETE FROM accounts WHERE id = ?"
	_, err := db.conn.Exec(sql, id)

	if err != nil {
		log.Errorf("Error deleting social account: %v", err)
		return fmt.Errorf("error deleting social account: %v", err)
	}

	return nil
}

func (db *DB) CreatePost(postStatus *controlv1.PostStatus, socialAccountId string) error {
	if db.conn == nil {
		return fmt.Errorf("database connection is not established")
	}

	sql := "INSERT INTO posts (id, social_account, status, created) VALUES (?, ?, ?, now())"
	_, err := db.conn.Exec(sql, postStatus.Id, socialAccountId, postStatus.Success)

	if err != nil {
		log.Errorf("Error inserting post: %v", err)
		return fmt.Errorf("error inserting post: %v", err)
	}

	return err
}

func (db *DB) SelectPosts() ([]*controlv1.PostStatus, error) {
	if db.conn == nil {
		return nil, fmt.Errorf("database connection is not established")
	}

	sql := "SELECT id, social_account, status, created FROM posts"
	rows, err := db.conn.Query(sql)

	if err != nil {
		log.Errorf("Error querying posts: %v", err)
		return nil, fmt.Errorf("error querying posts: %v", err)
	}

	defer rows.Close()

	var posts []*controlv1.PostStatus

	for rows.Next() {
		var post controlv1.PostStatus
		var socialAccountId string

		if err := rows.Scan(&post.Id, &socialAccountId, &post.Success, &post.Created); err != nil {
			log.Errorf("Error scanning post: %v", err)
			continue
		}

		post.SocialAccountId = socialAccountId
		posts = append(posts, &post)
	}

	return posts, nil
}
