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
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/ghlogin"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestListToken(t *testing.T) {
	assert := require.New(t)

	store := sqlstore.NewMemoryStore()

	tokenService := newTokenService(store)
	assert.NotNil(tokenService)

	// Do unauthenticated request. This returns error.
	res, err := tokenService.ListTokens(context.Background(), &apipb.ListTokenRequest{})
	assert.Nil(res)
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	user, token := createUserAndToken(assert, model.AuthConnectID, store)
	// Requests with a token will fail since token access isn't allowed for
	// this method.
	md := metadata.New(map[string]string{tokenHeaderName: token})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	res, err = tokenService.ListTokens(ctx, &apipb.ListTokenRequest{})
	assert.Nil(res)
	assert.Error(err)
	assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

	// Inject a session context for Connect and a sesson profile from GitHub
	// to emulate a logged-in user. The session profile is injected via the
	// various handlers that wraps the user. Both are set as value objects on
	// the context.
	dummyConnectSession := goconnect.Session{UserID: user.ExternalID}
	ctx = context.WithValue(context.Background(), goconnect.SessionContext, dummyConnectSession)

	res, err = tokenService.ListTokens(ctx, &apipb.ListTokenRequest{})
	assert.NotNil(res)
	assert.NoError(err)

	assert.NotNil(res.Tokens)
	assert.Len(res.Tokens, 1)

	// Repeat but with unknown user ID in the connect ID session. Should return
	// an error
	dummyConnectSession = goconnect.Session{UserID: user.ExternalID + "foo"}
	ctx = context.WithValue(context.Background(), goconnect.SessionContext, dummyConnectSession)

	res, err = tokenService.ListTokens(ctx, &apipb.ListTokenRequest{})
	assert.Nil(res)
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Repeat the test with a GitHub login
	user, token = createUserAndToken(assert, model.AuthGitHub, store)
	dummyGithubSession := ghlogin.Profile{
		Login: user.ExternalID,
	}
	ctx = context.WithValue(context.Background(), ghlogin.GitHubSessionProfile, dummyGithubSession)

	res, err = tokenService.ListTokens(ctx, &apipb.ListTokenRequest{})
	assert.NotNil(res)
	assert.NoError(err)
	assert.Len(res.Tokens, 1)
	assert.Equal(token, res.Tokens[0].Token.Value)
}

func TestCreateToken(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	tokenService := newTokenService(store)
	assert.NotNil(tokenService)

	// Ensure we authenticate
	res, err := tokenService.CreateToken(context.Background(), &apipb.Token{Resource: &wrappers.StringValue{Value: "/"}})
	assert.Error(err)
	assert.Nil(res)

	user, _, ctx := createAuthenticatedContext(assert, store)

	res, err = tokenService.CreateToken(ctx, &apipb.Token{Resource: &wrappers.StringValue{Value: "/"}})
	assert.Error(err)
	assert.Nil(res)

	// Auth with GH context
	dummyGithubSession := ghlogin.Profile{
		Login: user.ExternalID,
	}
	ctx = context.WithValue(context.Background(), ghlogin.GitHubSessionProfile, dummyGithubSession)

	// Create a new token
	res, err = tokenService.CreateToken(ctx, &apipb.Token{
		Write:    &wrappers.BoolValue{Value: true},
		Resource: &wrappers.StringValue{Value: "/"},
		Tags: map[string]string{
			"Hello": "There",
			"Name":  "Value",
			"Foo":   "This is the <script>alert();</script>",
		},
	})
	assert.NoError(err)
	assert.NotNil(res)
	assert.Equal("/", res.Resource.Value)
	assert.True(res.Write.Value)

	// Nil request fails
	res, err = tokenService.CreateToken(ctx, nil)
	assert.Error(err)
	assert.Nil(res)

	// Missing resource yields error
	res, err = tokenService.CreateToken(ctx, &apipb.Token{
		Write: &wrappers.BoolValue{Value: true},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Missing write key - error
	res, err = tokenService.CreateToken(ctx, &apipb.Token{
		Resource: &wrappers.StringValue{Value: "/"},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Invalid tag key or value will fail
	res, err = tokenService.CreateToken(ctx, &apipb.Token{
		Write:    &wrappers.BoolValue{Value: true},
		Resource: &wrappers.StringValue{Value: "/"},
		Tags: map[string]string{
			"": "Hmm",
		}})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
}

func TestDeleteToken(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	tokenService := newTokenService(store)
	assert.NotNil(tokenService)

	// Ensure we authenticate
	res, err := tokenService.DeleteToken(context.Background(), &apipb.DeleteTokenRequest{Token: &wrappers.StringValue{Value: "foo"}})
	assert.Error(err)
	assert.Nil(res)

	// Create GH login context
	user, token, ctx := createAuthenticatedContext(assert, store)

	res, err = tokenService.DeleteToken(ctx, &apipb.DeleteTokenRequest{Token: &wrappers.StringValue{Value: "bar"}})
	assert.Error(err)
	assert.Nil(res)

	// Auth with GH context
	dummyGithubSession := ghlogin.Profile{
		Login: user.ExternalID,
	}
	ctx = context.WithValue(context.Background(), ghlogin.GitHubSessionProfile, dummyGithubSession)

	// Remove the token
	res, err = tokenService.DeleteToken(ctx, &apipb.DeleteTokenRequest{
		Token: &wrappers.StringValue{Value: token},
	})
	assert.NoError(err)
	assert.NotNil(res)

	// Token should be removed from database
	_, err = store.RetrieveToken(token)
	assert.Equal(storage.ErrNotFound, err)

	// Deleting it twice should return Not Found error
	// Create a new token
	_, err = tokenService.DeleteToken(ctx, &apipb.DeleteTokenRequest{
		Token: &wrappers.StringValue{Value: token},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Request with nil value returns error
	_, err = tokenService.DeleteToken(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Request with no token returns error
	_, err = tokenService.DeleteToken(ctx, &apipb.DeleteTokenRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

}

func TestRetrieveToken(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	tokenService := newTokenService(store)
	assert.NotNil(tokenService)

	// Ensure we authenticate
	res, err := tokenService.RetrieveToken(context.Background(), &apipb.TokenRequest{Token: &wrappers.StringValue{Value: "foo"}})
	assert.Error(err)
	assert.Nil(res)

	user, token, ctx := createAuthenticatedContext(assert, store)

	res, err = tokenService.RetrieveToken(ctx, &apipb.TokenRequest{Token: &wrappers.StringValue{Value: "bar"}})
	assert.Error(err)
	assert.Nil(res)

	// Auth with GH context
	dummyGithubSession := ghlogin.Profile{
		Login: user.ExternalID,
	}
	ctx = context.WithValue(context.Background(), ghlogin.GitHubSessionProfile, dummyGithubSession)

	// Retrieve the token
	res, err = tokenService.RetrieveToken(ctx, &apipb.TokenRequest{
		Token: &wrappers.StringValue{Value: token},
	})

	assert.NoError(err)
	assert.NotNil(res)
	assert.Equal(token, res.Token.Value)

	// Empty token or nil request returns error
	res, err = tokenService.RetrieveToken(ctx, nil)
	assert.Error(err)
	assert.Nil(res)

	// Unknown token returns not found error
	_, err = tokenService.RetrieveToken(ctx, &apipb.TokenRequest{Token: &wrappers.StringValue{Value: "foo"}})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())
}

func TestUpdateToken(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	tokenService := newTokenService(store)
	assert.NotNil(tokenService)

	// Ensure we authenticate
	res, err := tokenService.UpdateToken(context.Background(), &apipb.Token{Token: &wrappers.StringValue{Value: "foo"}})
	assert.Error(err)
	assert.Nil(res)

	user, token, ctx := createAuthenticatedContext(assert, store)

	res, err = tokenService.UpdateToken(ctx, &apipb.Token{Token: &wrappers.StringValue{Value: "bar"}})
	assert.Error(err)
	assert.Nil(res)

	// Auth with GH context
	dummyGithubSession := ghlogin.Profile{
		Login: user.ExternalID,
	}
	ctx = context.WithValue(context.Background(), ghlogin.GitHubSessionProfile, dummyGithubSession)

	// Update the token
	res, err = tokenService.UpdateToken(ctx, &apipb.Token{
		Token:    &wrappers.StringValue{Value: token},
		Write:    &wrappers.BoolValue{Value: false},
		Resource: &wrappers.StringValue{Value: "/something/else"},
		Tags: map[string]string{
			"Hello": "There",
		},
	})
	assert.NoError(err)
	assert.NotNil(res)

	assert.False(res.Write.Value)
	assert.Equal("/something/else", res.Resource.Value)

	// The token should be updated in the store
	td, err := store.RetrieveToken(token)
	assert.NoError(err)
	assert.Equal(td.Resource, res.Resource.Value)
	assert.Equal(td.Write, res.Write.Value)

	// Use invalid resource
	_, err = tokenService.UpdateToken(ctx, &apipb.Token{
		Token:    &wrappers.StringValue{Value: token},
		Write:    &wrappers.BoolValue{Value: false},
		Resource: &wrappers.StringValue{Value: "   "},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Update a token that doesn't exist
	_, err = tokenService.UpdateToken(ctx, &apipb.Token{
		Token:    &wrappers.StringValue{Value: token + "somethingelse"},
		Write:    &wrappers.BoolValue{Value: false},
		Resource: &wrappers.StringValue{Value: "/something/else"},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Do a nil update
	_, err = tokenService.UpdateToken(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Update a token for another user. Should return not found
	_, otherToken := createUserAndToken(assert, model.AuthGitHub, store)
	_, err = tokenService.UpdateToken(ctx, &apipb.Token{
		Token:    &wrappers.StringValue{Value: otherToken},
		Resource: &wrappers.StringValue{Value: "/"},
		Write:    &wrappers.BoolValue{Value: true},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Invalid tag name
	res, err = tokenService.UpdateToken(ctx, &apipb.Token{
		Token: &wrappers.StringValue{Value: token},
		Tags: map[string]string{
			"": "<script>alert('Fishy alert');</script>",
		}})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

}

func TestTokenTags(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	tokenService := newTokenService(store)
	assert.NotNil(tokenService)

	_, token, ctx := createAuthenticatedContext(assert, store)

	doTagTests(ctx, assert, "", false, token, tagFunctions{
		ListTags:   tokenService.ListTokenTags,
		UpdateTag:  tokenService.UpdateTokenTag,
		GetTag:     tokenService.GetTokenTag,
		DeleteTag:  tokenService.DeleteTokenTag,
		UpdateTags: tokenService.UpdateTokenTags,
	})
}
