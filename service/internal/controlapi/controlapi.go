package controlapi

import (
	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"net/http"

	"github.com/rs/cors"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"
	buildinfo "github.com/jamesread/japella/internal/buildinfo"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/connectorcontroller"
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
)

type ControlApi struct {
	dbconn *sql.DB

	cc *connectorcontroller.ConnectionController
}

func (s *ControlApi) Start(cfg *runtimeconfig.CommonConfig) {
	if runtimeconfig.Get().Database.Enabled {
		s.reconnectDatabase(cfg.Database)
	}

	s.cc = connectorcontroller.New()

	log.Infof("ControlApi started with s: %+v", s)
}

func (s *ControlApi) reconnectDatabase(db runtimeconfig.DatabaseConfig) {
	url := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true", db.User, db.Password, db.Host, db.Database)

	var err error

	s.dbconn, err = sql.Open("mysql", url)

	if err != nil {
		log.Warnf("Failed to connect to database: %v", err)
	}
}

func (s *ControlApi) GetCannedPosts(ctx context.Context, req *connect.Request[controlv1.GetCannedPostsRequest]) (*connect.Response[controlv1.GetCannedPostsResponse], error) {
	if s.dbconn == nil {
		log.Warnf("Database connection is not established, cannot fetch canned posts")
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database connection is not established"))
	}

	sql := "SELECT id, content, created FROM canned_posts ORDER BY created DESC"
	rows, err := s.dbconn.Query(sql)

	if err != nil {
		log.Errorf("Error querying canned posts: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query canned posts: %w", err))
	}

	defer rows.Close()

	var posts []*controlv1.CannedPost

	for rows.Next() {
		var id, content, created string

		if err := rows.Scan(&id, &content, &created); err != nil {
			log.Errorf("Error scanning canned post: %v", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan canned post: %w", err))
		}
		posts = append(posts, &controlv1.CannedPost{
			Id:        id,
			Content:   content,
			CreatedAt: created,
		})
	}
	if err := rows.Err(); err != nil {
		log.Errorf("Error iterating over canned posts: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("error iterating over canned posts: %w", err))
	}

	// If no posts found, return an empty response
	if len(posts) == 0 {
		log.Warnf("No canned posts found in the database")
	}

	// Create response with canned GetCannedPosts
	// posts, if any

	res := connect.NewResponse(&controlv1.GetCannedPostsResponse{
		Posts: posts,
	})

	return res, nil
}

func (s *ControlApi) GetStatus(ctx context.Context, req *connect.Request[controlv1.GetStatusRequest]) (*connect.Response[controlv1.GetStatusResponse], error) {
	res := connect.NewResponse(&controlv1.GetStatusResponse{
		Status:       "OK!",
		Nanoservices: nanoservice.GetNanoservices(),
		Version:      buildinfo.Version,
	})

	return res, nil
}

func marshalPostingServices(cc *connectorcontroller.ConnectionController) []*controlv1.PostingService {
	services := make([]*controlv1.PostingService, 0)

	for id, svc := range cc.GetServices() {
		srv := &controlv1.PostingService{
			Id:       id,
			Identity: svc.GetIdentity(),
			Protocol: svc.GetProtocol(),
		}

		services = append(services, srv)
	}

	return services
}

func (s *ControlApi) SubmitPost(ctx context.Context, req *connect.Request[controlv1.SubmitPostRequest]) (*connect.Response[controlv1.SubmitPostResponse], error) {
	res := connect.NewResponse(&controlv1.SubmitPostResponse{})

	log.Infof("Received post request: %+v %+v %+v", req.Msg.Content, req.Msg.PostingService, s)

	postingService := s.cc.Get(req.Msg.PostingService)

	// If postingService implements connector.ConnectorWithWall, we can post to the wall
	if wallService, ok := postingService.(connector.ConnectorWithWall); ok {
		err := wallService.PostToWall(req.Msg.Content)

		if err != nil {
			log.Errorf("Error posting to wall: %v", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to post to wall: %w", err))
		}
	}

	return res, nil
}

func withCors(h http.Handler) http.Handler {
	middleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
	})

	return middleware.Handler(h)
}

func GetNewHandler() (string, http.Handler) {
	server := &ControlApi{}
	server.Start(runtimeconfig.Get())

	path, handler := controlv1connect.NewJapellaControlApiServiceHandler(server)

	return path, withCors(handler)
}

func (s *ControlApi) GetPostingServices(ctx context.Context, req *connect.Request[controlv1.GetPostingServicesRequest]) (*connect.Response[controlv1.GetPostingServicesResponse], error) {
	res := connect.NewResponse(&controlv1.GetPostingServicesResponse{
		Services: marshalPostingServices(s.cc),
	})

	return res, nil
}

func (s *ControlApi) CreateCannedPost(ctx context.Context, req *connect.Request[controlv1.CreateCannedPostRequest]) (*connect.Response[controlv1.CreateCannedPostResponse], error) {
	log.Infof("Creating canned post: %+v", req.Msg)

	sql := "INSERT INTO canned_posts (content) VALUES (?)"

	if s.dbconn == nil {
		log.Warnf("Database connection is not established, cannot create canned post")
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database connection is not established"))
	}

	_, err := s.dbconn.Exec(sql, req.Msg.Content)

	if err != nil {
		log.Errorf("Error inserting canned post: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert canned post: %w", err))
	}

	log.Infof("Canned post created successfully: %s", req.Msg.Content)

	res := connect.NewResponse(&controlv1.CreateCannedPostResponse{
		Message: "OK",
	})

	return res, nil
}

func (s *ControlApi) DeleteCannedPost(ctx context.Context, req *connect.Request[controlv1.DeleteCannedPostRequest]) (*connect.Response[controlv1.DeleteCannedPostResponse], error) {
	log.Infof("Deleting canned post with ID: %s", req.Msg.Id)

	sql := "DELETE FROM canned_posts WHERE id = ?"

	if s.dbconn == nil {
		log.Warnf("Database connection is not established, cannot delete canned post")
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database connection is not established"))
	}

	_, err := s.dbconn.Exec(sql, req.Msg.Id)

	if err != nil {
		log.Errorf("Error deleting canned post: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete canned post: %w", err))
	}

	log.Infof("Canned post deleted successfully with ID: %s", req.Msg.Id)

	res := connect.NewResponse(&controlv1.DeleteCannedPostResponse{
		Message: "OK",
	})

	return res, nil
}
