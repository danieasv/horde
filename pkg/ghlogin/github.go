package ghlogin

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
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ExploratoryEngineering/logging"
)

const (
	// This is the default scopes requested from GitHub
	defaultScopes = "user:read user:email"
	// GithubAuthCookieName is the name of the cookie used to store the session ID
	GithubAuthCookieName = "ee_github_session"
	// checkInterval is the frequency of session checks towards GitHub
	checkInterval = time.Minute * 5
	// sessionLength is the initial length of each session. It's much longer
	// than the check interval to spread the number of checks out
	sessionLength = checkInterval * 12
)

// gitHubSessionType is a type used to store the profile in the request context
type gitHubSessionType string

// GitHubSessionProfile is the value for the profile in the context
const GitHubSessionProfile = gitHubSessionType("GitHubSession")

// Authenticator is the GH OAuth interface. Create this to enable authentication
type Authenticator struct {
	Config   Config // Config is the authenticator's configuration
	sessions SessionStore
	client   http.Client
}

// New creates a new GH authenticator instance
func New(config Config) (*Authenticator, error) {

	sessionStore, err := NewSQLSessionStore(config.DBDriver, config.DBConnectionString)
	if err != nil {
		return nil, err
	}
	return &Authenticator{
		Config:   config,
		sessions: sessionStore,
		client:   http.Client{},
	}, nil
}

// Handler returns a handler for the (local) GitHub resource
func (a *Authenticator) Handler() http.Handler {
	return a
}

// AuthHandlerFunc returns a http.HandlerFunc wrapped in an authentication handler.
// If the user isn't authenticated it will return a 401 response, otherwise the wrapped function will be executed.
func (a *Authenticator) AuthHandlerFunc(funcToWrap http.HandlerFunc) http.HandlerFunc {
	return a.newAuthHandler(funcToWrap, a.sessions).ServeHTTP
}

const (
	loginPath    = "/login"
	callbackPath = "/callback"
	logoutPath   = "/logout"
)

func (a *Authenticator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, loginPath) {
		a.startLogin(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, callbackPath) {
		a.handleCallback(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, logoutPath) {
		a.startLogout(w, r)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(fmt.Sprintf("Don't know how to handle %s", r.URL.String())))
}

// makeState creates a state token for the OAuth service. It's just a random
// string of bytes
func (a *Authenticator) newState() string {
	buf := make([]byte, 32)
	rand.Read(buf)
	return hex.EncodeToString(buf)
}

// startLogin starts a login roundtrip to the GH OAuth server
func (a *Authenticator) startLogin(w http.ResponseWriter, r *http.Request) {
	// Create and persist state
	state := a.newState()
	if err := a.sessions.PutState(state); err != nil {
		logging.Error("Unable to persist state for OAuth token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error generating random state for session"))
		return
	}

	redirectURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		url.QueryEscape(a.Config.ClientID),
		url.QueryEscape(a.Config.CallbackURL),
		url.QueryEscape(defaultScopes),
		state)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// handleCallback handles the callback from the GH OAuth server
func (a *Authenticator) handleCallback(w http.ResponseWriter, r *http.Request) {
	requestState := r.URL.Query().Get("state")

	if err := a.sessions.RemoveState(requestState); err != nil {
		// Unknown state
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Unknown state in callback. Please try logging in again"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Did not get an access token code. Please try logging in again"))
		return
	}

	tokenURL := fmt.Sprintf("https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s&redirect_url=%s&state=%s",
		a.Config.ClientID,
		a.Config.ClientSecret,
		code,
		a.Config.CallbackURL,
		requestState)

	req, err := http.NewRequest(http.MethodPost, tokenURL, nil)
	if err != nil {
		logging.Error("Got error creating new request to %s: %v", tokenURL, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to request access token"))
		return
	}

	req.Header.Add("Accept", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		logging.Error("Error requesting access token from %s: %v", tokenURL, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to request access token"))
		return
	}
	if resp.StatusCode != http.StatusOK {
		logging.Error("Got error response from %s: %v", tokenURL, resp.Status)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Could not read response from remote server"))
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logging.Error("Got error reading response body from %s: %v", tokenURL, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to read response from remote server"))
		return
	}

	respData := make(map[string]interface{})
	if err := json.Unmarshal(buf, &respData); err != nil {
		logging.Error("Malformed response from remote server: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to read response from remote server"))
		return
	}
	// Check if the error response is set
	if err, ok := respData["error"]; ok {
		logging.Warning("Got error response from GitHub: %s (req = %s)", err, tokenURL)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("I'm not configured correctly. Sorry about that."))
		return
	}
	tmp, ok := respData["access_token"]
	if !ok {
		logging.Error("Did not get the expected return. Got %+v", respData)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("I made poo poo"))
		return
	}
	token := tmp.(string)

	userURL := "https://api.github.com/user"
	req, err = http.NewRequest(http.MethodGet, userURL, nil)
	if err != nil {
		logging.Error("Error creating request to %s: %v", userURL, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to request profile from remote server"))
		return
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("token %s", token))
	resp, err = a.client.Do(req)
	if err != nil {
		logging.Error("Error performing request to %s: %v", userURL, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to request profile from remote server"))
		return
	}
	if resp.StatusCode != http.StatusOK {
		logging.Error("Request to %s returned %v", userURL, resp.Status)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to request profile from remote server"))
		return
	}

	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		logging.Error("Error reading response body from %s: %v", userURL, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to request profile from remote server"))
		return
	}

	userProfile := make(map[string]interface{})
	if err := json.Unmarshal(buf, &userProfile); err != nil {
		logging.Error("Error unmarshaling response body from %s: %v", userURL, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Unable to request profile from remote server"))
		return
	}

	profile := profileFromMap(userProfile)

	expires := time.Now().UnixNano() + int64(sessionLength)
	// Create session. The access token never expires but the access might be revoked by
	// the user. If the user has revoked the access token disable the session. The session
	// check is done separately
	sessionID := newSessionID()
	if err := a.sessions.CreateSession(sessionID, token, expires, profile); err != nil {
		logging.Error("Could not create session for user %+v: %v", profile, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Profile could not be stored"))
		return
	}

	cookie := &http.Cookie{
		Name:     GithubAuthCookieName,
		Value:    sessionID,
		HttpOnly: true,
		MaxAge:   0,
		Path:     "/",
		Secure:   a.Config.SecureCookie,
	}
	http.SetCookie(w, cookie)

	http.Redirect(w, r, a.Config.LoginSuccess, http.StatusTemporaryRedirect)

}

// startLogout does a logout roundtrip. On the client side this is just to
// remove the auth cookie and the token from the session store
func (a *Authenticator) startLogout(w http.ResponseWriter, r *http.Request) {
	// Remove session cookie, delete tokens
	cookie, err := r.Cookie(GithubAuthCookieName)
	if err != nil {
		logging.Warning("Got error retrieving cookie: %v", err)
		http.Redirect(w, r, a.Config.LogoutSuccess, http.StatusTemporaryRedirect)
		return
	}

	sessionID := cookie.Value
	session, err := a.sessions.GetSession(sessionID, time.Now().UnixNano())
	if err == nil {
		if err := a.sessions.RemoveSession(sessionID); err != nil {
			logging.Warning("Couldn't remove session: %v", err)
		}
		// Remove access token from github
		removeURL := fmt.Sprintf("https://api.github.com/applications/%s/grant", a.Config.ClientID)
		body := strings.NewReader(fmt.Sprintf(`{"access_token":"%s"}`, session.AccessToken))
		req, err := http.NewRequest(http.MethodDelete, removeURL, body)
		if err == nil {
			req.SetBasicAuth(a.Config.ClientID, a.Config.ClientSecret)
			_, err := a.client.Do(req)
			if err != nil {
				logging.Warning("Couldn't remove access token: %v", err)
			}
		}
	}
	removeCookie(w)
	http.Redirect(w, r, a.Config.LogoutSuccess, http.StatusTemporaryRedirect)
}

// NewAuthHandler returns a http.Handler that requires authentication. If the request
// isn't authenticated a 401 Unauthorized is returned to the client, otherwise the
// existing http.Handler is invoked. The Session object is passed along in the request's
// Context.
func (a *Authenticator) newAuthHandler(existingHandler http.Handler, sessions SessionStore) http.Handler {
	return &authHandler{existingHandler: existingHandler, sessions: sessions}
}

// Profile reads the user profile from the http.Request context
func (a *Authenticator) Profile(r *http.Request) (Profile, error) {
	p := getProfileFromContext(r)
	if p == nil {
		return Profile{}, errors.New("not logged in")
	}
	return *p, nil
}

// StartSessionChecker launches a profile checker goroutine
func (a *Authenticator) StartSessionChecker() {
	go func() {
		client := http.Client{}
		for {
			time.Sleep(checkInterval)
			timeToCheck := time.Now().UnixNano() + int64(checkInterval)
			profiles, err := a.sessions.GetSessions(timeToCheck)
			if err != nil {
				logging.Warning("Got error checking for expired sessions: %v", err)
				continue
			}
			for _, v := range profiles {
				// Check profile status
				checkURL := fmt.Sprintf("https://api.github.com/applications/%s/token",
					a.Config.ClientID)
				body := strings.NewReader(fmt.Sprintf(`{"access_token":"%s"}`, v.AccessToken))
				req, err := http.NewRequest("POST", checkURL, body)
				if err != nil {
					logging.Warning("Got error creating GitHub session check request: %v", err)
					continue
				}
				req.SetBasicAuth(a.Config.ClientID, a.Config.ClientSecret)
				resp, err := client.Do(req)
				if err != nil {
					logging.Warning("Got error from GitHub token check: %v", err)
					continue
				}

				// Read the entire response, then discard. This ensures the http.Client is
				// reused properly.
				io.Copy(ioutil.Discard, resp.Body)
				resp.Body.Close()

				if resp.StatusCode != 200 {
					if err := a.sessions.RemoveSession(v.SessionID); err != nil {
						logging.Warning("Unable to remove session %v: %v", v.SessionID, err)
					}
					logging.Debug("Removed GitHub session with ID %s (status code=%d:%s)", v.SessionID, resp.StatusCode, resp.Status)
					continue
				}
				if err := a.sessions.RefreshSession(v.SessionID, int64(sessionLength)); err != nil {
					logging.Warning("Unable to update session %v: %v", v.SessionID, err)
				}
			}
		}
	}()
}
