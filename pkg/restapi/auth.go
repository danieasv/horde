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
	"net/http"
	"strings"

	"github.com/TelenorDigital/goconnect"
	"github.com/gorilla/websocket"

	"github.com/ExploratoryEngineering/logging"
	"github.com/ExploratoryEngineering/rest"
	"github.com/eesrc/horde/pkg/api"
	"github.com/eesrc/horde/pkg/ghlogin"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

const tokenHeader = "X-API-Token"
const tokenParam = "api_token"

// newAuthHandler dispatches authentication to one of the three authentication
// handlers -- check for tokens, check for CONNECT ID and check for GitHub auth.
// These are all expensive, so check for cookies and headers before invoking
// them. They also assume that they're the final authority so they'll return
// 401 if it doesn't work.
func (s *restServer) newAuthHandler(connectHandler http.HandlerFunc, githubHandler http.HandlerFunc, tokenHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Test token first. It's the quickest auth check
		token := r.Header.Get(tokenHeader)
		if token == "" && websocket.IsWebSocketUpgrade(r) {
			token = r.URL.Query().Get(tokenParam)
		}

		if token != "" {
			tokenHandler(w, r)
			metrics.DefaultCoreCounters.AuthTokenCount.Add(1)
			return
		}
		ghCookie, err := r.Cookie(ghlogin.GithubAuthCookieName)
		if err == nil && ghCookie != nil && ghCookie.Value != "" {
			// Test the GH auth
			githubHandler(w, r)
			metrics.DefaultCoreCounters.AuthGithubCount.Add(1)
			return
		}
		// Test the CONNECT ID handler last since we don't know what the cookie is called.
		connectHandler(w, r)

		if goconnect.HasSessionCookie(r) {
			metrics.DefaultCoreCounters.AuthConnectCount.Add(1)
		}
	}
}

func (s *restServer) emulateConnect(f http.HandlerFunc) http.HandlerFunc {
	logging.Warning("Emulating CONNECT ID")
	return func(w http.ResponseWriter, r *http.Request) {
		// Make sure there's no CONNECT ID sessions here. Terminate if it exists since
		// this means we might be running on a production service with connect disabled
		sess := r.Context().Value(goconnect.SessionContext)
		if sess != nil {
			logging.Error("Did not expect CONNECT ID session to exist but got %+v", sess)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		connectID := r.Header.Get("X-CONNECT-ID")
		if connectID == "computersaysno" {
			logging.Error("COMPUTER SAYS NO!!!!1!!")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if connectID == "" {
			connectID = "1"
			logging.Debug("X-CONNECT-ID not set. Auto-assigning 1. use X-CONNECT-ID to override user ID")
		}
		emulatedSession := goconnect.Session{
			UserID:        connectID,
			Name:          "Mr Robot",
			Locale:        "en-gb",
			Email:         "doe" + connectID + "@example.com",
			VerifiedEmail: true,
			Phone:         "555-999-88",
			VerifiedPhone: true}
		newContext := context.WithValue(r.Context(), goconnect.SessionContext, emulatedSession)
		newContext = context.WithValue(newContext, api.AuthKey, model.AuthConnectID)
		f(w, r.WithContext(newContext))
		metrics.DefaultCoreCounters.AuthConnectCount.Add(1)
	}
}

func (s *restServer) createConnectHandler(cc ConnectIDParameters, secureCookie bool, existingHandler http.HandlerFunc) http.HandlerFunc {
	if !cc.Enabled {
		return existingHandler
	}
	if cc.Emulate {
		return s.emulateConnect(existingHandler)
	}

	logging.Debug("CONNECT ID is enabled")
	cconfig := cc.ConnectConfig()
	cconfig.UseSecureCookie = secureCookie
	logging.Debug("CONNECT ID Config = %+v", cconfig)

	var connect *goconnect.GoConnect
	if cc.sessionStore != nil {
		logging.Info("Using persistent CONNECT ID session store")
		cstore, err := goconnect.NewSQLStorage(cc.sessionStore)
		if err != nil {
			logging.Error("Unable to create session store for CONNECT ID. Falling back to memory-backed store")
			cstore = goconnect.NewMemoryStorage()
		}
		connect = goconnect.NewConnectIDWithStorage(cconfig, cstore)
	} else {
		logging.Warning("Using memory-based CONNECT ID sesison store")
		connect = goconnect.NewConnectID(cconfig)
	}
	// This isn't very pretty
	s.mux.Handle("/connect/", connect.Handler())
	s.mux.HandleFunc("/connect/profile", rest.AddCORSHeaders(connect.SessionProfile))

	return connect.NewAuthHandlerFunc(existingHandler)
}

func (s *restServer) createGitHubHandler(ghConfig ghlogin.Config, secureCookie bool, existingHandler http.HandlerFunc) http.HandlerFunc {
	logging.Debug("GitHub OAuth is enabled")

	ghConfig.SecureCookie = secureCookie
	var err error
	s.githubAuth, err = ghlogin.New(ghConfig)
	if err != nil {
		logging.Error("Unable to create GitHub auth handler. Returning a deny-all-handler: %v", err)
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Authentication is disabled", http.StatusForbidden)
		}
	}
	s.githubAuth.StartSessionChecker()
	s.mux.Handle("/github/", s.githubAuth.Handler())
	return s.githubAuth.AuthHandlerFunc(existingHandler)
}

// Handler for the API tokens.
func (s *restServer) createTokenHandler(handler http.HandlerFunc, tokenstore storage.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenVal := r.Header.Get(tokenHeader)
		// WebSockets are special; tokens might be supplied through query parameters for web sockets
		// since headers aren't supported for WebSockets. These api tokens *will* show up in the
		// request log but we impose the restriction that they must be read-only and limit access to
		// just the websocket data.
		wsTokenInQuery := false
		if tokenVal == "" && websocket.IsWebSocketUpgrade(r) {
			tokenVal = r.URL.Query().Get(tokenParam)
			wsTokenInQuery = true
		}

		token, err := tokenstore.RetrieveToken(tokenVal)
		if err != nil {
			if err == storage.ErrNotFound {
				reportError(w, http.StatusUnauthorized, "Unknown API token", nil)
				return
			}
			logging.Warning("Unable to retrieve token %s: %v", tokenVal, err)
			reportError(w, http.StatusInternalServerError, "Unable to process request", nil)
			return
		}

		// Stop writable tokens in query -- see above
		if wsTokenInQuery && token.Write {
			reportError(w, http.StatusBadRequest, "Tokens in WebSockets must be read-only", nil)
			return
		}
		// Check if the token matches the request
		path := r.URL.Path
		method := r.Method
		// Ensure the token matches the path
		if !strings.HasPrefix(path, token.Resource) {
			reportError(w, http.StatusForbidden, "Access denied", nil)
			return
		}

		// Block POST, DELETE and GET unless it is a write operation
		if !token.Write && (method == http.MethodPatch ||
			method == http.MethodPost || method == http.MethodDelete) {
			reportError(w, http.StatusForbidden, "Access denied", nil)
			return
		}

		// Finally, retrieve the user and add it to the context
		user, err := tokenstore.RetrieveUser(token.UserID)
		if err != nil {
			if err == storage.ErrNotFound {
				reportError(w, http.StatusUnauthorized, "Access denied", nil)
				return
			}
			logging.Warning("Unable to retrieve user for token %s: %v", tokenVal, err)
			reportError(w, http.StatusInternalServerError, "Unable to read user", nil)
			return
		}
		newContext := context.WithValue(r.Context(), api.UserKey, &user)
		newContext = context.WithValue(newContext, api.AuthKey, model.AuthToken)
		handler(w, r.WithContext(newContext))
	}
}
