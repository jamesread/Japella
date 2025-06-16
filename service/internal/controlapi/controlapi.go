package controlapi

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"golang.org/x/oauth2"

	"net/http"

	"github.com/rs/cors"

	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"
	buildinfo "github.com/jamesread/japella/internal/buildinfo"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/connectorcontroller"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	log "github.com/sirupsen/logrus"
	"os"

	"github.com/google/uuid"
)

type ControlApi struct {
	db *db.DB

	oauth2states map[string]*oauth2State

	cc *connectorcontroller.ConnectionController

	socialaccounts map[string]*connector.SocialAccount
}

type oauth2State struct {
	config    *oauth2.Config
	connector connector.OAuth2Connector
	verifier  string
}

func (s *ControlApi) Start(cfg *runtimeconfig.CommonConfig) {
	s.db = &db.DB{}
	s.db.ReconnectDatabase(cfg.Database)

	s.oauth2states = make(map[string]*oauth2State)
	s.cc = connectorcontroller.New(s.db)

	log.Infof("ControlApi started with s: %+v", s)

	s.loadSocialAccounts()
}

func (s *ControlApi) loadSocialAccounts() {
	s.socialaccounts = make(map[string]*connector.SocialAccount)

	for _, account := range s.db.SelectSocialAccounts() {
		s.socialaccounts[account.Id] = &connector.SocialAccount{
			Id:         account.Id,
			Connector:  account.Connector,
			Identity:   account.Identity,
			OAuthToken: account.OAuthToken,
		}
	}
}

func (s *ControlApi) GetCannedPosts(ctx context.Context, req *connect.Request[controlv1.GetCannedPostsRequest]) (*connect.Response[controlv1.GetCannedPostsResponse], error) {
	cannedPosts := s.db.SelectCannedPosts()

	ret := make([]*controlv1.CannedPost, 0, len(cannedPosts))

	for _, post := range cannedPosts {
		ret = append(ret, &controlv1.CannedPost{
			Id:      post.Id,
			Content: post.Content,
		})
	}

	res := connect.NewResponse(&controlv1.GetCannedPostsResponse{
		Posts: ret,
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

func (s *ControlApi) marshalSocialAccounts() []*controlv1.SocialAccount {
	accounts := make([]*controlv1.SocialAccount, 0)

	for account := range s.socialaccounts {
		socialAccount := s.socialaccounts[account]

		connectorService := s.cc.Get(socialAccount.Connector)

		accounts = append(accounts, &controlv1.SocialAccount{
			Id:        socialAccount.Id,
			Connector: socialAccount.Connector,
			Identity:  socialAccount.Identity,
			Icon:      connectorService.GetIcon(),
		})
	}

	return accounts
}

func (s *ControlApi) SubmitPost(ctx context.Context, req *connect.Request[controlv1.SubmitPostRequest]) (*connect.Response[controlv1.SubmitPostResponse], error) {
	res := &controlv1.SubmitPostResponse{}

	log.Infof("Received post request for social accounts: %+v", req.Msg.SocialAccounts)

	for _, accountId := range req.Msg.SocialAccounts {
		log.Infof("Processing post for account: %s", accountId)

		socialAccount := s.socialaccounts[accountId]

		if socialAccount == nil {
			log.Errorf("Social account not found: %s", accountId)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found: %s", accountId))
		}

		postingService := s.cc.Get(socialAccount.Connector)

		if postingService == nil {
			log.Errorf("Posting service not found for connector: %s", socialAccount.Connector)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("posting service not found for connector: %s", socialAccount.Connector))
		}

		if wallService, ok := postingService.(connector.ConnectorWithWall); ok {
			log.Infof("Posting to wall service wit account id: %v with is of connection proto: %v", accountId, wallService.GetProtocol())

			postResult := wallService.PostToWall(socialAccount, req.Msg.Content)

			if postResult.Err != nil {
				log.Errorf("Error posting to wall: %v", postResult.Err)
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to post to wall: %w", postResult.Err))
			}

			res.PostUrl = postResult.URL
			res.Success = true
		} else {
			log.Warnf("Posting service does not support wall posting: %s", postingService.GetProtocol())
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("posting service does not support wall posting: %s", postingService.GetProtocol()))
		}
	}

	return connect.NewResponse(res), nil
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

func GetNewHandler() (string, http.Handler, *ControlApi) {
	server := &ControlApi{}
	server.Start(runtimeconfig.Get())

	path, handler := controlv1connect.NewJapellaControlApiServiceHandler(server)

	return path, withCors(handler), server
}

func (s *ControlApi) GetSocialAccounts(ctx context.Context, req *connect.Request[controlv1.GetSocialAccountsRequest]) (*connect.Response[controlv1.GetSocialAccountsResponse], error) {
	s.loadSocialAccounts()

	res := connect.NewResponse(&controlv1.GetSocialAccountsResponse{
		Accounts: s.marshalSocialAccounts(),
	})

	return res, nil
}

func (s *ControlApi) CreateCannedPost(ctx context.Context, req *connect.Request[controlv1.CreateCannedPostRequest]) (*connect.Response[controlv1.CreateCannedPostResponse], error) {
	log.Infof("Creating canned post: %+v", req.Msg)

	s.db.CreateCannedPost(req.Msg.Content)

	res := connect.NewResponse(&controlv1.CreateCannedPostResponse{
		Message: "OK",
	})

	return res, nil
}

func (s *ControlApi) DeleteCannedPost(ctx context.Context, req *connect.Request[controlv1.DeleteCannedPostRequest]) (*connect.Response[controlv1.DeleteCannedPostResponse], error) {
	s.db.DeleteCannedPost(req.Msg.Id)

	res := connect.NewResponse(&controlv1.DeleteCannedPostResponse{
		Message: "OK",
	})

	return res, nil
}

func (s *ControlApi) GetConnectors(ctx context.Context, req *connect.Request[controlv1.GetConnectorsRequest]) (*connect.Response[controlv1.GetConnectorsResponse], error) {
	log.Infof("Fetching connectors")

	res := connect.NewResponse(&controlv1.GetConnectorsResponse{
		Connectors: marshalConnectors(s.cc, req.Msg.OnlyWantOauth),
	})

	return res, nil
}

func marshalConnectors(cc *connectorcontroller.ConnectionController, onlyWantOauth bool) []*controlv1.Connector {
	services := make([]*controlv1.Connector, 0)

	for id, svc := range cc.GetServices() {
		_, isOAuth := svc.(connector.OAuth2Connector)

		if !isOAuth && onlyWantOauth {
			log.Infof("Skipping connector %s as it does not support OAuth", id)
			continue
		}

		srv := &controlv1.Connector{
			Id:       id,
			Name:     svc.GetProtocol(),
			Icon:     svc.GetIcon(),
			HasOauth: isOAuth,
		}

		services = append(services, srv)
	}

	log.Infof("Marshalled connectors: %+v", len(services))

	return services
}

func (s *ControlApi) StartOAuth(ctx context.Context, req *connect.Request[controlv1.StartOAuthRequest]) (*connect.Response[controlv1.StartOAuthResponse], error) {
	log.Infof("Starting OAuth flow for connector: %s", req.Msg.ConnectorId)

	svc := s.cc.Get(req.Msg.ConnectorId)

	if svc == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("connector not found: %s", req.Msg.ConnectorId))
	}

	oauthConnector, ok := svc.(connector.OAuth2Connector)

	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("connector does not support OAuth: %s", req.Msg.ConnectorId))
	}

	verifier := oauth2.GenerateVerifier()

	cfg := oauthConnector.GetOAuth2Config()

	stateKey := uuid.New().String()

	url := cfg.AuthCodeURL(stateKey, oauth2.S256ChallengeOption(verifier))

	s.oauth2states[stateKey] = &oauth2State{
		config:    cfg,
		connector: oauthConnector,
		verifier:  verifier,
	}

	log.Infof("OAuth flow started for connector: %s, config: %+v", req.Msg.ConnectorId, cfg)
	log.Infof("OAuth URL: %s", url)

	res := connect.NewResponse(&controlv1.StartOAuthResponse{
		Url: url,
	})

	return res, nil
}

func (c *ControlApi) registerAccount(connector string, accessToken string) {
	err := c.db.RegisterAccount(connector, accessToken)

	if err != nil {
		log.Errorf("Error registering account: %v", err)
	} else {
		log.Infof("Account registered successfully with connector: %s", accessToken)
	}
}

func (s *ControlApi) OAuth2CallbackHandler(w http.ResponseWriter, r *http.Request) {
	log.Infof("Received OAuth2 callback with URL: %s", r.URL.String())

	errText := r.URL.Query().Get("error")

	if errText != "" {
		redirect(w, fmt.Sprintf("OAuth2 error: %v", errText), "bad")
		return
	}

	stateKey := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	state := s.oauth2states[stateKey]

	if state == nil {
		redirect(w, fmt.Sprintf("state not found: %v", stateKey), "bad")
		return
	}

	client := &http.Client{
		Transport: utils.NewLoggingTransport(nil),
	}

	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	token, err := state.config.Exchange(ctx, code, oauth2.VerifierOption(state.verifier))

	if err != nil {
		log.Errorf("Error exchanging OAuth2 code: %v", err)
		redirect(w, "error exchanging OAuth2 code", "bad")
		return
	}

	log.Infof("Received token on exchange: %+v", token)

	log.Infof("State connector: %+v", state)

	s.registerAccount(state.connector.GetProtocol(), token.AccessToken)

	log.Infof("OAuth2 access token: %s", token.AccessToken)

	redirect(w, "Account registered successfully", "good")
}

func redirect(w http.ResponseWriter, message string, msgType string) {
	inDev := os.Getenv("JAPELLA_VITE") == "true"

	server := "http://localhost:8080"

	if inDev {
		server = "http://localhost:5173"
	}

	url := fmt.Sprintf("%v/?notification=%v&type=%v", server, message, msgType)

	w.Header().Set("Location", url)
	w.Write([]byte(fmt.Sprintf("<html><body><h1>Redirecting...</h1><p>%s</p><a href = \"%v\">click here</a></body></html>", message, url)))

	log.Infof("Redirecting with message: %v", message)
}

func (s *ControlApi) DeleteSocialAccount(ctx context.Context, req *connect.Request[controlv1.DeleteSocialAccountRequest]) (*connect.Response[controlv1.DeleteSocialAccountResponse], error) {
	log.Infof("Deleting social account with ID: %s", req.Msg.Id)

	s.db.DeleteSocialAccount(req.Msg.Id)
	delete(s.socialaccounts, req.Msg.Id)

	res := connect.NewResponse(&controlv1.DeleteSocialAccountResponse{
		StandardResponse: &controlv1.StandardResponse{
			Success: true,
			Message: "OK",
		},
	})

	return res, nil
}

func (s *ControlApi) RefreshSocialAccount(ctx context.Context, req *connect.Request[controlv1.RefreshSocialAccountRequest]) (*connect.Response[controlv1.RefreshSocialAccountResponse], error) {
	log.Infof("Refreshing social account with ID: %s", req.Msg.Id)

	socialAccount := s.socialaccounts[req.Msg.Id]

	if socialAccount == nil {
		log.Errorf("Social account not found: %s", req.Msg.Id)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found: %s", req.Msg.Id))
	}

	connectorService := s.cc.Get(socialAccount.Connector)

	if connectorService == nil {
		log.Errorf("Connector service not found for connector: %s", socialAccount.Connector)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("connector service not found for connector: %s", socialAccount.Connector))
	}

	connectorService.OnRefresh(socialAccount)

	res := connect.NewResponse(&controlv1.RefreshSocialAccountResponse{
		StandardResponse: &controlv1.StandardResponse{
			Success: true,
		},
	})

	return res, nil
}
