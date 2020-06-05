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
	"testing"

	"github.com/TelenorDigital/goconnect"
	"github.com/eesrc/horde/pkg/ghlogin"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestGRPCAuth(t *testing.T) {
	assert := require.New(t)

	store := sqlstore.NewMemoryStore()

	// Empty context will return nil, ie no auth
	ctx := context.Background()
	assert.Nil(gRPCAuth(ctx, store))

	// Set the Connect session in the state. It should return nil since the
	// user doesn't exist
	session := goconnect.Session{
		UserID: "connect-user-1",
		Email:  "johndoe@example.com",
	}
	ctx = context.WithValue(context.Background(), goconnect.SessionContext, session)
	assert.Nil(gRPCAuth(ctx, store))

	// Create the user in the backend store. Since the user exists in the
	// backend store we have an authenticated user
	team := model.NewTeam()
	team.ID = store.NewTeamID()
	user := model.NewUser(store.NewUserID(), session.UserID, model.AuthConnectID, team.ID)
	assert.NoError(store.CreateUser(user, team))

	auth := gRPCAuth(ctx, store)
	assert.NotNil(auth)
	assert.Equal(user.ID, auth.User.ID)
	assert.Equal(model.AuthConnectID, auth.Method)

	// Add a token to the user. The token header is added as a metadata by the
	// gRPC gateway so a token header should authenticate as well.
	token := model.NewToken()
	token.GenerateToken()
	token.Resource = "/"
	token.UserID = user.ID
	token.Write = true
	assert.NoError(store.CreateToken(token))

	ctx = metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{tokenHeaderName: token.Token}))
	auth = gRPCAuth(ctx, store)
	assert.NotNil(auth)
	assert.Equal(user.ID, auth.User.ID)
	assert.Equal(model.AuthToken, auth.Method)

	// Create a GitHub authenticated user in the backend
	team = model.NewTeam()
	team.ID = store.NewTeamID()
	profile := ghlogin.Profile{
		Login: "gh-login-1",
	}
	ctx = context.WithValue(context.Background(), ghlogin.GitHubSessionProfile, profile)

	// This should fail since the user does not exist in the store
	assert.Nil(gRPCAuth(ctx, store))

	// Create user, auth should work
	user = model.NewUser(store.NewUserID(), profile.Login, model.AuthGitHub, team.ID)
	assert.NoError(store.CreateUser(user, team))

	auth = gRPCAuth(ctx, store)
	assert.NotNil(auth)
	assert.Equal(user.ID, auth.User.ID)
	assert.Equal(model.AuthGitHub, auth.Method)
}
