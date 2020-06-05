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
	"context"
	"errors"
	"net/http"
	"time"
)

type authHandler struct {
	existingHandler http.Handler
	sessions        SessionStore
}

func removeCookie(w http.ResponseWriter) {
	rmCookie := &http.Cookie{
		Name:     GithubAuthCookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, rmCookie)

}

func getProfileFromContext(r *http.Request) *Profile {
	val := r.Context().Value(GitHubSessionProfile)
	if val == nil {
		return nil
	}
	ret, ok := val.(*Profile)
	if !ok {
		return nil
	}
	return ret
}

func getProfileFromCookie(w http.ResponseWriter, r *http.Request, sessionStore SessionStore) (Profile, error) {
	cookie, err := r.Cookie(GithubAuthCookieName)
	if cookie == nil || err == http.ErrNoCookie || cookie.Value == "" {
		return Profile{}, errors.New("no session cookie")
	}
	sessionID := cookie.Value
	if err != nil {
		removeCookie(w)
		return Profile{}, err
	}
	sess, err := sessionStore.GetSession(sessionID, time.Now().UnixNano())
	if err != nil {
		removeCookie(w)
		return Profile{}, err
	}
	return sess.Profile, nil
}

func (a *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		a.existingHandler.ServeHTTP(w, r)
		return
	}

	profile, err := getProfileFromCookie(w, r, a.sessions)
	if err != nil {
		http.Error(w, "You are not authorized to view this page. Try logging in again.", http.StatusUnauthorized)
		return
	}
	a.existingHandler.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), GitHubSessionProfile, &profile)))
}
