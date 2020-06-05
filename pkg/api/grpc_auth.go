package api

//
// Copyright 2020 Telenor Digital AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
import (
	"context"

	"github.com/ExploratoryEngineering/logging"
	"github.com/TelenorDigital/goconnect"
	"github.com/eesrc/horde/pkg/ghlogin"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const tokenHeaderName = "x-api-token"

// Keys for request contexts
type contextKey string

// UserKey is the context key for the user object in the context
const UserKey = contextKey("user")

// AuthKey is the context key for the authentication method in the context
const AuthKey = contextKey("auth")

// TODO(stalehd): Move dependency. Remove context use.

// The default gRPC auth implementation for tags
type defaultGrpcAuth struct {
	Store storage.DataStore
}

func (d *defaultGrpcAuth) EnsureAuth(ctx context.Context) (*authResult, error) {
	auth := gRPCAuth(ctx, d.Store)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "Must authenticate")
	}
	return auth, nil
}

type authResult struct {
	User           model.User
	Method         model.AuthMethod
	Token          string
	ConnectSession goconnect.Session
	GitHubProfile  ghlogin.Profile
}

// gRPCAuth authenticates a request through the gRPC API.
func gRPCAuth(ctx context.Context, store storage.DataStore) *authResult {
	// Token might be part of the metadata if it is a request received via the
	// gRPC gateway. If not this might be the context itself.
	//
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		t := md.Get(tokenHeaderName)
		if len(t) == 1 {
			// ok - there's a token. Find the corresponding user
			token, err := store.RetrieveToken(t[0])
			if err != nil {
				if err != storage.ErrNotFound {
					logging.Warning("Error retrieving token %s: %v", t[0], err)
				}
				return nil
			}
			user, err := store.RetrieveUser(token.UserID)
			if err != nil {
				if err != storage.ErrNotFound {
					logging.Warning("Error retrieving user for token %s (user id=%d): %v", token.Token, token.UserID, err)
				}
				return nil
			}
			return &authResult{
				User:   user,
				Method: model.AuthToken,
				Token:  token.Token,
			}
		}
	}
	am := ctx.Value(AuthKey)
	au := ctx.Value(UserKey)
	if am != nil && au != nil {
		user, uok := au.(*model.User)
		meth, mok := am.(model.AuthMethod)
		if uok && mok {
			return &authResult{
				User:   *user,
				Method: meth,
			}
		}
	}

	// If the context has a goconnect session object we're authenticated via
	// a session cookie
	sc := ctx.Value(goconnect.SessionContext)
	if sc != nil {
		session, ok := sc.(goconnect.Session)
		if !ok {
			// The value isn't what we expect - just return
			return nil
		}
		user, err := store.RetrieveUserByExternalID(session.UserID, model.AuthConnectID)
		if err != nil {
			if err != storage.ErrNotFound {
				logging.Warning("Error retrieving user for connect user ID: %s: %v", session.UserID, err)
			}
			return nil
		}
		return &authResult{
			User:           *user,
			Method:         model.AuthConnectID,
			ConnectSession: session,
		}
	}
	sc = ctx.Value(ghlogin.GitHubSessionProfile)
	if sc != nil {
		profile, ok := sc.(ghlogin.Profile)
		if !ok {
			// Not the type we expected - just return
			return nil
		}
		user, err := store.RetrieveUserByExternalID(profile.Login, model.AuthGitHub)
		if err != nil {
			if err != storage.ErrNotFound {
				logging.Warning("Error retrieving user for github login %s: %v", profile.Login, err)
			}
			return nil
		}
		return &authResult{
			User:          *user,
			Method:        model.AuthGitHub,
			GitHubProfile: profile,
		}
	}

	// No authentication token or sessions found
	return nil
}
