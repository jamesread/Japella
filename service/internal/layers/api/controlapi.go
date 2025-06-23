package api

import (
	"context"
	"fmt"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"
	"golang.org/x/oauth2"

	"net/http"

	"github.com/rs/cors"

	"github.com/jamesread/golure/pkg/redact"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"
	buildinfo "github.com/jamesread/japella/internal/buildinfo"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/connectorcontroller"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/layers/authentication"
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	log "github.com/sirupsen/logrus"
	"os"

	"github.com/google/uuid"
)

type ControlApi struct {
	DB *db.DB

	oauth2states map[string]*oauth2State

	statusMessages []*controlv1.StatusMessage

	cc *connectorcontroller.ConnectionController
}

type oauth2State struct {
	config    *oauth2.Config
	connector connector.OAuth2Connector
	verifier  string
}

func (s *ControlApi) Start(cfg *runtimeconfig.CommonConfig) {
	s.statusMessages = make([]*controlv1.StatusMessage, 0)

	s.DB = &db.DB{}
	err := s.DB.ReconnectDatabase(cfg.Database)

	if err != nil {
		s.statusMessages = append(s.statusMessages, &controlv1.StatusMessage{
			Message: fmt.Sprintf("Critical database error: %v", err),
			Type:    "error",
		})

		log.Errorf("Database startup problem: %v", err)

		return
	}

	s.oauth2states = make(map[string]*oauth2State)
	s.cc = connectorcontroller.New(s.DB)

	log.Infof("ControlAPI started")
}

func (s *ControlApi) GetCannedPosts(ctx context.Context, req *connect.Request[controlv1.GetCannedPostsRequest]) (*connect.Response[controlv1.GetCannedPostsResponse], error) {
	cannedPosts := s.DB.SelectCannedPosts()

	ret := make([]*controlv1.CannedPost, 0, len(cannedPosts))

	for _, post := range cannedPosts {
		ret = append(ret, &controlv1.CannedPost{
			Id:      post.ID,
			Content: post.Content,
			CreatedAt: post.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	res := connect.NewResponse(&controlv1.GetCannedPostsResponse{
		Posts: ret,
	})

	return res, nil
}

func (s *ControlApi) getAuthenticatedUser(ctx context.Context) *authentication.AuthenticatedUser {
	v := authn.GetInfo(ctx)

	if v == nil {
		return nil
	} else {
		return v.(*authentication.AuthenticatedUser)
	}
}

func (s *ControlApi) GetStatus(ctx context.Context, req *connect.Request[controlv1.GetStatusRequest]) (*connect.Response[controlv1.GetStatusResponse], error) {
	var authenticatedUser *authentication.AuthenticatedUser
	username := ""

	if authenticatedUser = s.getAuthenticatedUser(ctx); authenticatedUser != nil {
		username = authenticatedUser.User.Username
	}

	log.Infof("GetStatus called by user: %s", username)

	res := connect.NewResponse(&controlv1.GetStatusResponse{
		Status:       "OK!",
		Nanoservices: nanoservice.GetNanoservices(),
		Version:      buildinfo.Version,
		Username:     username,
		IsLoggedIn:   authenticatedUser != nil,
		StatusMessages: s.statusMessages,
	})

	return res, nil
}

func (s *ControlApi) marshalSocialAccounts(onlyActive bool) []*controlv1.SocialAccount {
	accounts := make([]*controlv1.SocialAccount, 0)

	for _, socialAccount := range s.DB.SelectSocialAccounts(onlyActive) {
		connectorService := s.cc.Get(socialAccount.Connector)

		accounts = append(accounts, &controlv1.SocialAccount{
			Id:        socialAccount.ID,
			Connector: socialAccount.Connector,
			Identity:  socialAccount.Identity,
			Icon:      connectorService.GetIcon(),
			Active:    socialAccount.Active,
		})
	}

	return accounts
}

func (s *ControlApi) SubmitPost(ctx context.Context, req *connect.Request[controlv1.SubmitPostRequest]) (*connect.Response[controlv1.SubmitPostResponse], error) {
	res := &controlv1.SubmitPostResponse{}

	log.Infof("Received post request for social accounts: %+v", req.Msg.SocialAccounts)

	for _, accountId := range req.Msg.SocialAccounts {
		log.Infof("Processing post for account: %v", accountId)

		postStatus := &controlv1.PostStatus{
			Success:   false,
			SocialAccountId: accountId,
			Content:  req.Msg.Content,
		}

		socialAccount, _ := s.DB.GetSocialAccount(accountId)

		s.tryPostStatus(req.Msg.Content, socialAccount, postStatus)

		s.DB.CreatePost(&db.Post{
			SocialAccountID: socialAccount.ID,
			Content:         req.Msg.Content,
			Status:         postStatus.Success,
			PostURL:         postStatus.PostUrl,
		})

		res.Posts = append(res.Posts, postStatus)
	}

	return connect.NewResponse(res), nil
}

func (s *ControlApi) tryPostStatus(content string, socialAccount *db.SocialAccount, postStatus *controlv1.PostStatus) {
	postingService := s.cc.Get(socialAccount.Connector)

	if postingService == nil {
		log.Errorf("Posting service not found for connector: %s", socialAccount.Connector)
		return
	}

	postStatus.SocialAccountIcon = postingService.GetIcon()
	postStatus.SocialAccountIdentity = socialAccount.Identity

	if wallService, ok := postingService.(connector.ConnectorWithWall); ok {
		log.Infof("Posting to wall service wit account id: %v with is of connection proto: %v", socialAccount.ID, wallService.GetProtocol())

		postResult := wallService.PostToWall(toConnectorSA(socialAccount), content)

		if postResult.Err != nil {
			log.Errorf("Error posting to wall: %v", postResult.Err)
			return
		}

		postStatus.PostUrl = postResult.URL
		postStatus.Success = true

	} else {
		log.Warnf("Posting service does not support wall posting: %s", postingService.GetProtocol())
		return
	}
}

func toConnectorSA(socialAccount *db.SocialAccount) *connector.SocialAccount {
	return &connector.SocialAccount{
		Id:         socialAccount.ID,
		Connector:  socialAccount.Connector,
		Identity:   socialAccount.Identity,
		OAuthToken: socialAccount.OAuth2Token,
	}
}

func withCors(h http.Handler) http.Handler {
	opts := cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
	}

	opts.ExposedHeaders = append(opts.ExposedHeaders, "Set-Cookie")

	middleware := cors.New(opts)

	return middleware.Handler(h)
}

func GetNewHandler() (string, http.Handler, *ControlApi) {
	server := &ControlApi{}
	server.Start(runtimeconfig.Get())

	path, handler := controlv1connect.NewJapellaControlApiServiceHandler(server)

	return path, withCors(handler), server
}

func (s *ControlApi) GetSocialAccounts(ctx context.Context, req *connect.Request[controlv1.GetSocialAccountsRequest]) (*connect.Response[controlv1.GetSocialAccountsResponse], error) {
	res := connect.NewResponse(&controlv1.GetSocialAccountsResponse{
		Accounts: s.marshalSocialAccounts(req.Msg.OnlyActive),
	})

	return res, nil
}

func (s *ControlApi) CreateCannedPost(ctx context.Context, req *connect.Request[controlv1.CreateCannedPostRequest]) (*connect.Response[controlv1.CreateCannedPostResponse], error) {
	log.Infof("Creating canned post: %+v", req.Msg)

	err := s.DB.CreateCannedPost(req.Msg.Content)

	if err != nil {
		log.Errorf("Error creating canned post: %v", err)

		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create canned post: %w", err))
	} else {
		log.Infof("Canned post created successfully")

		res := connect.NewResponse(&controlv1.CreateCannedPostResponse{
			Message: "OK",
		})

		return res, nil
	}
}

func (s *ControlApi) DeleteCannedPost(ctx context.Context, req *connect.Request[controlv1.DeleteCannedPostRequest]) (*connect.Response[controlv1.DeleteCannedPostResponse], error) {
	err := s.DB.DeleteCannedPost(req.Msg.Id)

	if err != nil {
		log.Errorf("Error deleting canned post: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete canned post: %w", err))
	} else {
		res := connect.NewResponse(&controlv1.DeleteCannedPostResponse{
			Message: "OK",
		})

		return res, nil
	}
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
			Name:     svc.GetProtocol(),
			Icon:     svc.GetIcon(),
			HasOauth: isOAuth,
		}

		cfgProvider, isConfigProvider := svc.(connector.ConfigProvider)

		if isConfigProvider {
			srv.Issues = cfgProvider.CheckConfiguration().Issues
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

	err = s.DB.RegisterAccount(&db.SocialAccount{
		Connector:  state.connector.GetProtocol(),
		OAuth2Token:  token.AccessToken,
		OAuth2TokenExpiry: token.Expiry,
		OAuth2RefreshToken: token.RefreshToken,
	})

	if err != nil {
		log.Errorf("Error registering account: %v", err)
		redirect(w, fmt.Sprintf("Error registering account: %v", err), "bad")
	} else {
		redirect(w, fmt.Sprintf("Successfully registered account for connector: %s", state.connector.GetProtocol()), "good")
	}
}

func redirect(w http.ResponseWriter, message string, msgType string) {
	inDev := os.Getenv("JAPELLA_DEV_REDIRECT_VITE") == "true"

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

	err := s.DB.DeleteSocialAccount(req.Msg.Id)

	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete social account: %w", err))
	} else {
		res := connect.NewResponse(&controlv1.DeleteSocialAccountResponse{
			StandardResponse: &controlv1.StandardResponse{
				Success: true,
				Message: "OK",
			},
		})

		return res, nil
	}
}

func (s *ControlApi) RefreshSocialAccount(ctx context.Context, req *connect.Request[controlv1.RefreshSocialAccountRequest]) (*connect.Response[controlv1.RefreshSocialAccountResponse], error) {
	log.Infof("Refreshing social account with ID: %s", req.Msg.Id)

	socialAccount, _ := s.DB.GetSocialAccount(req.Msg.Id)

	if socialAccount == nil {
		log.Errorf("Social account not found: %s", req.Msg.Id)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("social account not found: %s", req.Msg.Id))
	}

	connectorService := s.cc.Get(socialAccount.Connector)

	if connectorService == nil {
		log.Errorf("Connector service not found for connector: %s", socialAccount.Connector)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("connector service not found for connector: %s", socialAccount.Connector))
	}

	connectorService.OnRefresh(socialAccount);

	res := connect.NewResponse(&controlv1.RefreshSocialAccountResponse{
		StandardResponse: &controlv1.StandardResponse{
			Success: true,
		},
	})

	return res, nil
}

func (s *ControlApi) GetTimeline(ctx context.Context, req *connect.Request[controlv1.GetTimelineRequest]) (*connect.Response[controlv1.GetTimelineResponse], error) {
	posts, err := s.DB.SelectPosts()

	if err != nil {
		log.Errorf("Error selecting posts: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to retrieve posts: %w", err))
	}

	timeline := make([]*controlv1.PostStatus, 0, len(posts))


	for _, post := range posts {
		socialAccountIcon := "mdi:question-mark-circle"
		socialAccountIdentity := "Unknown"

		if post.SocialAccount != nil {
			socialAccountIcon = s.cc.Get(post.SocialAccount.Connector).GetIcon()
			socialAccountIdentity = post.SocialAccount.Identity
		}

		timeline = append(timeline, &controlv1.PostStatus{
			Id:             post.ID,
			Created:      post.CreatedAt.Format("2006-01-02 15:04:05"),
			SocialAccountId: post.SocialAccountID,
			SocialAccountIcon: socialAccountIcon,
			SocialAccountIdentity: socialAccountIdentity,
			Content:         post.Content,
			Success:         post.Status,
			PostUrl:         post.PostURL,
		})
	}

	res := connect.NewResponse(&controlv1.GetTimelineResponse{
		Posts: timeline,
	})

	return res, nil
}

func (s *ControlApi) SetSocialAccountActive(ctx context.Context, req *connect.Request[controlv1.SetSocialAccountActiveRequest]) (*connect.Response[controlv1.SetSocialAccountActiveResponse], error) {
	log.Infof("Setting social account active state for ID: %v to %v", req.Msg.Id, req.Msg.Active)

	err := s.DB.SetSocialAccountActive(req.Msg.Id, req.Msg.Active)

	if err != nil {
		log.Errorf("Error setting social account active state: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to set social account active state: %w", err))
	}

	res := connect.NewResponse(&controlv1.SetSocialAccountActiveResponse{
		StandardResponse: &controlv1.StandardResponse{
			Success: true,
			Message: "OK",
		},
	})

	return res, nil
}

func (s *ControlApi) LoginWithUsernameAndPassword(ctx context.Context, req *connect.Request[controlv1.LoginWithUsernameAndPasswordRequest]) (*connect.Response[controlv1.LoginWithUsernameAndPasswordResponse], error) {
	log.Infof("Login with username and password for user: %s", req.Msg.Username)

	user := s.DB.GetUserByUsername(req.Msg.Username)

	if user == nil {
		log.Warnf("User not found: %s", req.Msg.Username)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user not found: %s", req.Msg.Username))
	}

	if user.PasswordHash == "" {
		log.Warnf("User has no password set: %s", req.Msg.Username)
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user has no password set: %s", req.Msg.Username))
	}

	res := connect.NewResponse(&controlv1.LoginWithUsernameAndPasswordResponse{})

	match, err := utils.VerifyPassword(user.PasswordHash, req.Msg.Password)

	if err != nil || !match {
		res.Msg.StandardResponse = &controlv1.StandardResponse{
			Success: false,
			Message: "Invalid username or password",
		}

		return res, nil
	} else {
		res.Msg.StandardResponse = &controlv1.StandardResponse{
			Success: true,
			Message: "Login successful",
		}
		res.Msg.Username = user.Username
	}


	sid := uuid.New().String()
	log.Infof("Creating session for user: %s with session ID: %s", user.Username, sid)

	// Create a session in the database
	err = s.DB.CreateSession(sid, user.ID)

	if err != nil {
		log.Errorf("Error creating session: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create session: %w", err))
	}

	c := http.Cookie{
		Name: "japella-sid",
		Value: sid,
		HttpOnly: true,
		Path: "/",
		SameSite: http.SameSiteStrictMode,
	}

	if os.Getenv("JAPELLA_SECURE_COOKIES") == "false" {
		c.Secure = false
	} else {
		c.Secure = true
	}

	log.Infof("Setting session cookie: %v", c.String())
	res.Header().Add("Set-Cookie", c.String())

	return res, nil
}

func (s *ControlApi) GetUsers(ctx context.Context, req *connect.Request[controlv1.GetUsersRequest]) (*connect.Response[controlv1.GetUsersResponse], error) {
	log.Infof("Fetching users")

	users, err := s.DB.SelectUsers()

	if err != nil {
		log.Errorf("Error selecting users: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to retrieve users: %w", err))
	}

	res := connect.NewResponse(&controlv1.GetUsersResponse{
		Users: make([]*controlv1.UserAccount, 0, len(users)),
	})

	for _, user := range users {
		res.Msg.Users = append(res.Msg.Users, &controlv1.UserAccount{
			Id:       user.ID,
			Username: user.Username,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return res, nil
}

func (s *ControlApi) GetApiKeys(ctx context.Context, req *connect.Request[controlv1.GetApiKeysRequest]) (*connect.Response[controlv1.GetApiKeysResponse], error) {
	log.Infof("Fetching API keys")

	apiKeys, err := s.DB.SelectAPIKeys()

	if err != nil {
		log.Errorf("Error selecting API keys: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to retrieve API keys: %w", err))
	}

	res := connect.NewResponse(&controlv1.GetApiKeysResponse{
		Keys: make([]*controlv1.ApiKey, 0, len(apiKeys)),
	})

	for _, key := range apiKeys {
		res.Msg.Keys = append(res.Msg.Keys, &controlv1.ApiKey{
			Id:        key.ID,
			KeyValue:  redact.RedactString(key.KeyValue),
			CreatedAt: key.CreatedAt.Format("2006-01-02 15:04:05"),
			UserId:    key.UserAccountID,
			Username:  key.UserAccount.Username,
		})
	}

	return res, nil
}

func (s *ControlApi) GetCvars(ctx context.Context, req *connect.Request[controlv1.GetCvarsRequest]) (*connect.Response[controlv1.GetCvarsResponse], error) {
	log.Infof("Fetching cvars")

	cvars, err := s.DB.SelectCvars()

	if err != nil {
		log.Errorf("Error selecting cvars: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to retrieve cvars: %w", err))
	}

	res := connect.NewResponse(&controlv1.GetCvarsResponse{
		CvarCategories: make(map[string]*controlv1.CvarCategory),
	})

	for _, cvar := range cvars {
		if cvar.Category == "" {
			cvar.Category = "Uncategorized"
		}

		category, exists := res.Msg.CvarCategories[cvar.Category]

		if !exists {
			category = &controlv1.CvarCategory{
				Cvars: make([]*controlv1.Cvar, 0),
			}

			category.Name = cvar.Category

			res.Msg.CvarCategories[cvar.Category] = category
		}

		category.Cvars = append(category.Cvars, &controlv1.Cvar{
			KeyName:      cvar.KeyName,
			ValueString:     cvar.ValueString,
			ValueInt:    cvar.ValueInt,
			Description: cvar.Description,
			Type:		 cvar.Type,
			Title:        cvar.Title,
		})
	}

	return res, nil
}

func (s *ControlApi) SaveUserPreferences(ctx context.Context, req *connect.Request[controlv1.SaveUserPreferencesRequest]) (*connect.Response[controlv1.SaveUserPreferencesResponse], error) {
	authenticatedUser := s.getAuthenticatedUser(ctx)

	if authenticatedUser == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	log.Infof("Saving user preferences for user: %s", authenticatedUser.User.Username)

	err := s.DB.SaveUserPreferences(&db.UserPreferences{
		UserAccountID: authenticatedUser.User.ID,
		Language: req.Msg.Language,
	})

	if err != nil {
		log.Errorf("Error saving user preferences: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save user preferences: %w", err))
	}

	res := connect.NewResponse(&controlv1.SaveUserPreferencesResponse{
		StandardResponse: &controlv1.StandardResponse{
			Success: true,
			Message: "Preferences saved successfully",
		},
	})

	return res, nil
}

func (s *ControlApi) CreateApiKey(ctx context.Context, req *connect.Request[controlv1.CreateApiKeyRequest]) (*connect.Response[controlv1.CreateApiKeyResponse], error) {
	authenticatedUser := s.getAuthenticatedUser(ctx)

	log.Infof("Creating API key for user: %s", authenticatedUser.User.Username)

	newKeyValue := uuid.New().String()

	apiKey, err := s.DB.CreateApiKey(authenticatedUser.User, newKeyValue)

	if err != nil {
		log.Errorf("Error creating API key: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create API key: %w", err))
	}

	res := connect.NewResponse(&controlv1.CreateApiKeyResponse{
		StandardResponse: &controlv1.StandardResponse {
			Success: true,
		},
		NewKeyValue: newKeyValue,
	})

	if err != nil {
		log.Errorf("Error creating API key: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create API key: %w", err))
	}

	log.Infof("API key created successfully: %s", apiKey.KeyValue)

	return res, nil
}

func (s *ControlApi) RevokeApiKey(ctx context.Context, req *connect.Request[controlv1.RevokeApiKeyRequest]) (*connect.Response[controlv1.RevokeApiKeyResponse], error) {
	err := s.DB.RevokeApiKey(req.Msg.Id)

	if err != nil {
		log.Errorf("Error revoking API key: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to revoke API key: %w", err))
	}

	res := connect.NewResponse(&controlv1.RevokeApiKeyResponse{
		StandardResponse: &controlv1.StandardResponse{
			Success: true,
			Message: "API key revoked successfully",
		},
	})

	return res, nil
}

func (s *ControlApi) SetCvar(ctx context.Context, req *connect.Request[controlv1.SetCvarRequest]) (*connect.Response[controlv1.SetCvarResponse], error) {
	cvar := s.DB.GetCvar(req.Msg.KeyName)

	if cvar == nil {
		log.Warnf("cvar not found: %s", req.Msg.KeyName)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("cvar not found: %s", req.Msg.KeyName))
	}

	var err error

	switch (cvar.Type) {
	case "password":
		fallthrough
	case "text":
		log.Infof("Setting cvar %s to string value: %s", cvar.KeyName, req.Msg.ValueString)
		err = s.DB.SetCvarString(cvar.KeyName, req.Msg.ValueString)
	case "int":
		err = s.DB.SetCvarInt(cvar.KeyName, req.Msg.ValueInt)
	default:
		err = fmt.Errorf("unsupported cvar type: %s", cvar.Type)
	}

	if err != nil {
		log.Errorf("Error setting cvar: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to set cvar: %w", err))
	}

	res := connect.NewResponse(&controlv1.SetCvarResponse{
		StandardResponse: &controlv1.StandardResponse{
			Success: true,
			Message: "cvar set successfully",
		},
	})

	return res, nil
}
