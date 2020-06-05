package restapi

//
//Copyright 2019 Telenor Digital AS
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//
import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/api"
	"github.com/eesrc/horde/pkg/api/apipb"
	"google.golang.org/grpc/metadata"

	"github.com/eesrc/horde/pkg/ghlogin"
	"github.com/eesrc/horde/pkg/utils/grpcutil"

	"github.com/ExploratoryEngineering/logging"
	"github.com/ExploratoryEngineering/rest"
	"github.com/TelenorDigital/goconnect"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/gorilla/handlers"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/crypto/acme/autocert"
)

// Server is the server interface for the REST API server
type Server interface {
	// Start launches the server
	Start() error
	// Stop stops the servers
	Stop() error
}

// Server is the type that handles the HTTP server serving the REST API.
type restServer struct {
	params          ServerParameters
	connect         ConnectIDParameters
	mux             *http.ServeMux
	server          *http.Server
	storage         storage.DataStore
	imageStore      storage.FirmwareImageStore
	sender          api.DownstreamMessageSender
	done            chan bool
	mgr             output.Manager
	deviceFieldMask model.FieldMaskParameters
	githubAuth      *ghlogin.Authenticator
	apiServer       apipb.HordeServer
}

// NewServer creates a new REST API server
func NewServer(params ServerParameters,
	dataClientParams grpcutil.GRPCClientParam,
	cc ConnectIDParameters,
	ghc ghlogin.Config,
	store storage.DataStore,
	imageStore storage.FirmwareImageStore,
	sender api.DownstreamMessageSender,
	mgr output.Manager,
	fieldMasks model.FieldMaskParameters) Server {

	ret := &restServer{
		params:          params,
		connect:         cc,
		storage:         store,
		imageStore:      imageStore,
		sender:          sender,
		mgr:             mgr,
		deviceFieldMask: fieldMasks}

	dcc, err := grpcutil.NewGRPCClientConnection(dataClientParams)
	if err != nil {
		logging.Error("Unable to create data store client: %v", err)
		return nil
	}
	dataStoreClient := datastore.NewDataStoreClient(dcc)

	ret.mux = http.NewServeMux()

	// Inject the gRPC handler with the existing handler. This is a bit of a mess
	// right now. TODO(stalehd): Clean up
	grpcMux, err := ret.grpcResourceHandlers(fieldMasks, ret.storage,
		dataStoreClient, ret.imageStore, mgr, sender)
	if err != nil {
		logging.Error("Unable to create gRPC gateway mux: %v", err)
		return nil
	}

	runtime.HTTPError = CustomHTTPError
	handler := ret.createWrappedMux(grpcMux)

	handler = ret.authSessionToUserHandlerFunc(handler)

	secureCookie := (params.ACME.Enabled || params.TLSCertFile != "")
	connectHandler := ret.createConnectHandler(cc, secureCookie, handler)
	githubHandler := ret.createGitHubHandler(ghc, secureCookie, handler)
	tokenHandler := ret.createTokenHandler(handler, store)

	handler = ret.newAuthHandler(connectHandler, githubHandler, tokenHandler)

	ret.mux.HandleFunc("/", ret.perfCounterHandler(rest.AddCORSHeaders(handler).ServeHTTP))
	ret.server = &http.Server{
		Addr:    params.Endpoint,
		Handler: handlers.CompressHandler(ret.addRequestLogging(params.InlineRequestlog, ret.mux)),
	}
	if params.ACME.Enabled {
		ret.server.Addr = ":https"
	}

	return ret
}

func (s *restServer) startServer(result chan error, tls bool, tlscert, tlskey string) {
	defer func() {
		s.done <- true
	}()

	if tls {
		logging.Info("REST API runs on port 443")
		s.server.Addr = ":https"
		if err := s.server.ListenAndServeTLS(s.params.TLSCertFile, s.params.TLSKeyFile); err != http.ErrServerClosed {
			result <- err
		}
		return
	}
	logging.Info("REST API runs on %s", s.params.Endpoint)
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		result <- err
	}
}

func (s *restServer) Start() error {
	result := make(chan error)
	go func(result chan error) {
		if s.params.ACME.Enabled {
			logging.Info("Using Let's Encrypt for certificates")
			// See https://godoc.org/golang.org/x/crypto/acme/autocert#example-Manager
			m := &autocert.Manager{
				Cache:      autocert.DirCache(s.params.ACME.SecretDir),
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(s.params.ACME.HostList()...),
			}
			go http.ListenAndServe(":http", m.HTTPHandler(nil))
			s.server.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}
			s.startServer(result, true, "", "")
			return
		}
		if s.params.TLSKeyFile != "" && s.params.TLSCertFile != "" {
			logging.Info("Using TLS configuration in %s/%s", s.params.TLSCertFile, s.params.TLSKeyFile)
			s.startServer(result, true, s.params.TLSCertFile, s.params.TLSKeyFile)
			return
		}
		s.startServer(result, false, "", "")
	}(result)
	select {
	case err := <-result:
		return err
	case <-time.After(100 * time.Millisecond):
		break
	}

	return nil
}

func (s *restServer) Stop() error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
	case <-s.done:
	}
	return nil
}

// UserFromRequest returns the ID of the currenly logged in user.
func (s *restServer) UserFromRequest(r *http.Request) *model.User {
	if r == nil || r.Context() == nil {
		return nil
	}
	v := r.Context().Value(api.UserKey)
	if v == nil {
		return nil
	}
	user, ok := v.(*model.User)
	if !ok {
		return nil
	}
	return user
}

func (s *restServer) grpcResourceHandlers(
	fieldMask model.FieldMaskParameters,
	dataStore storage.DataStore,
	dataStoreClient datastore.DataStoreClient,
	imageStore storage.FirmwareImageStore,
	outputManager output.Manager,
	messageSender api.DownstreamMessageSender) (*runtime.ServeMux, error) {

	// Start the system service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO(stalehd): Move into other section later and make gRPC service interface
	s.apiServer = api.NewHordeAPIService(dataStore, fieldMask, outputManager, dataStoreClient, messageSender, imageStore)

	gwmux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(name string) (string, bool) {
			if strings.ToLower(name) == "x-api-token" {
				return strings.ToLower(name), true
			}
			return "", false
		}),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: false}),
		runtime.WithMetadata(func(ctx context.Context, r *http.Request) metadata.MD {
			// Move authorization stuff into the metadata. We'll just use the api token
			// header for now. Cookies must be handled separately.
			// Include session cookes from Connect and GitHub later.
			return metadata.MD{
				"Token": []string{r.Header.Get("X-API-Token")},
			}
		}),
		runtime.WithMarshalerOption("application/json+pretty", &runtime.JSONPb{Indent: "  "}))
	if err := apipb.RegisterHordeHandlerServer(ctx, gwmux, s.apiServer); err != nil {
		return nil, err
	}
	return gwmux, nil
}

// requestLogWriter writes request logs to the default logger
type requestLogWriter struct {
	inline bool
}

func (r *requestLogWriter) Write(p []byte) (int, error) {
	if r.inline {
		logging.Info("REQUESTLOG: %s", string(p))
	}
	return len(p), nil
}

// addLogging is a simple wrapper that dumps the request to the debug log.
func (s *restServer) addRequestLogging(inline bool, h http.Handler) http.Handler {
	if inline {
		f := &requestLogWriter{inline: inline}
		return handlers.LoggingHandler(f, h)
	}
	// If the file doesn't exist, create it, or append to the file
	var f io.Writer
	var err error
	f, err = os.OpenFile(s.params.RequestLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logging.Warning("Unable to open request log in %s", s.params.RequestLog)
		f = &requestLogWriter{inline: inline}
	}
	return handlers.LoggingHandler(f, h)
}

// addOrUpdateConnectUser adds or updates the user in the backend store
func (s *restServer) addOrUpdateConnectUser(session goconnect.Session) *model.User {
	user, err := s.storage.RetrieveUserByExternalID(session.UserID, model.AuthConnectID)
	if err != nil && err != storage.ErrNotFound {
		logging.Warning("Unable to do user lookup for user with Connect ID %s: %v", session.UserID, err)
		return nil
	}
	if err == storage.ErrNotFound {
		t := model.NewTeam()
		t.ID = s.storage.NewTeamID()
		t.Tags.SetTag("name", "My private team")

		u := model.NewUser(s.storage.NewUserID(), session.UserID, model.AuthConnectID, t.ID)
		u.Email = session.Email
		u.Phone = session.Phone
		u.Name = session.Name
		u.VerifiedEmail = session.VerifiedEmail
		u.VerifiedPhone = session.VerifiedPhone

		t.AddMember(model.NewMember(u, model.AdminRole))

		if err := s.storage.CreateUser(u, t); err != nil {
			logging.Warning("Unable to store user: %v", err)
			return nil
		}
		logging.Debug("Created user: %+v", u)

		c := model.NewCollection()
		c.FieldMask = s.deviceFieldMask.DefaultFields()
		c.ID = s.storage.NewCollectionID()
		c.TeamID = t.ID
		c.Tags.SetTag("name", "My default collection")
		if err := s.storage.CreateCollection(u.ID, c); err != nil {
			logging.Warning("Unable to create a collection for a new user: %v", err)
			return nil
		}
		if err := s.storage.UpdateTeam(u.ID, t); err != nil {
			logging.Warning("Unable to update team for new user: %v", err)
			return nil
		}
		user = &u
	}
	if user != nil {
		if user.Name != session.Name ||
			user.Phone != session.Phone ||
			user.Email != session.Email ||
			user.VerifiedEmail != session.VerifiedEmail ||
			user.VerifiedPhone != session.VerifiedPhone {
			user.Name = session.Name
			user.Phone = session.Phone
			user.Email = session.Email
			user.VerifiedEmail = session.VerifiedEmail
			user.VerifiedPhone = session.VerifiedPhone
			if err := s.storage.UpdateUser(user); err != nil {
				logging.Warning("Unable to update user: %v", err)
			}
		}
	}
	return user
}

// addOrUpdateConnectUser adds or updates the user in the backend store
func (s *restServer) addOrUpdateGitHubUser(profile ghlogin.Profile) *model.User {
	user, err := s.storage.RetrieveUserByExternalID(profile.Login, model.AuthGitHub)
	if err != nil && err != storage.ErrNotFound {
		logging.Warning("Unable to do user lookup for user with GitHub login %s: %v", profile.Login, err)
		return nil
	}
	if err == storage.ErrNotFound {
		t := model.NewTeam()
		t.ID = s.storage.NewTeamID()
		t.Tags.SetTag("name", "My private team")

		u := model.NewUser(s.storage.NewUserID(), profile.Login, model.AuthGitHub, t.ID)
		u.Email = profile.Email
		u.Phone = ""
		u.Name = profile.Name
		u.VerifiedEmail = true
		u.VerifiedPhone = false
		u.AvatarURL = profile.AvatarURL

		t.AddMember(model.NewMember(u, model.AdminRole))

		if err := s.storage.CreateUser(u, t); err != nil {
			logging.Warning("Unable to store user: %v (user = %+v", err, u)
			return nil
		}
		logging.Debug("Created GH user: %+v", u)

		c := model.NewCollection()
		c.FieldMask = s.deviceFieldMask.DefaultFields()
		c.ID = s.storage.NewCollectionID()
		c.TeamID = t.ID
		c.Tags.SetTag("name", "My default collection")
		if err := s.storage.CreateCollection(u.ID, c); err != nil {
			logging.Warning("Unable to create a collection for a new user: %v", err)
			return nil
		}
		if err := s.storage.UpdateTeam(u.ID, t); err != nil {
			logging.Warning("Unable to update team for new user: %v", err)
			return nil
		}
		user = &u
	}
	if user != nil {
		if user.Name != profile.Name ||
			user.Email != profile.Email ||
			user.AvatarURL != profile.AvatarURL {

			user.Name = profile.Name
			user.Email = profile.Email
			user.AvatarURL = profile.AvatarURL

			if err := s.storage.UpdateUser(user); err != nil {
				logging.Warning("Unable to update user: %v", err)
			}
		}
	}
	return user
}

// authSessionToUserHandlerFunc grabs the Connect ID session from the context and
// injects the actual user information into the context.
func (s *restServer) authSessionToUserHandlerFunc(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newContext := r.Context()

		session := newContext.Value(goconnect.SessionContext)
		if session != nil {
			if sess, ok := session.(goconnect.Session); ok {
				if user := s.addOrUpdateConnectUser(sess); user != nil {
					newContext = context.WithValue(newContext, api.UserKey, user)
					newContext = context.WithValue(newContext, api.AuthKey, model.AuthConnectID)
				}
			}
		}
		if s.githubAuth != nil {
			profile, err := s.githubAuth.Profile(r)
			if err == nil {
				if user := s.addOrUpdateGitHubUser(profile); user != nil {
					newContext = context.WithValue(newContext, api.UserKey, user)
					newContext = context.WithValue(newContext, api.AuthKey, model.AuthGitHub)
				}
			}
		}
		f(w, r.WithContext(newContext))
	}
}
