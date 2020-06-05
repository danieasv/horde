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
	"strings"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type tokenService struct {
	store storage.DataStore
}

// newTokenService creates a new token service object that implements the
// TokensService interface.
func newTokenService(store storage.DataStore) tokenService {
	return tokenService{
		store: store,
	}
}

// Ensure user is logged in and is authenticated with a regular login. Returns
// nil if everything is OK. This is used both internally and the by the tag
// functions. This is slightly different from the default authentication since
// the tokens themselves are used in the default authentication scheme.
func (t *tokenService) EnsureAuth(ctx context.Context) (*authResult, error) {
	auth := gRPCAuth(ctx, t.store)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "Must authenticate")
	}
	if !auth.Method.Login() {
		return nil, status.Error(codes.PermissionDenied, "Must be logged in to manage tokens")
	}
	return auth, nil
}

// Utility method that retrieves a token. The error is ready to return to
// the gRPC client, ie it uses errors from the codes package.
func (t *tokenService) loadToken(auth *authResult, tv *wrappers.StringValue) (model.Token, error) {
	if tv == nil || strings.TrimSpace(tv.Value) == "" {
		return model.Token{}, status.Error(codes.InvalidArgument, "Missing token")
	}

	token, err := t.store.RetrieveToken(tv.Value)
	if err != nil {
		if err == storage.ErrNotFound {
			return model.Token{}, status.Error(codes.NotFound, "Unknown token")
		}
		logging.Warning("Error retrieving token: %v", err)
		return model.Token{}, status.Error(codes.Internal, "Unable to retrieve token to update")
	}
	if token.UserID != auth.User.ID {
		// This is another user. Return NotFound
		return model.Token{}, status.Error(codes.NotFound, "Unknown token")
	}
	return token, nil
}

// This is a wrapped loadToken for the tag functions
func (t *tokenService) LoadTaggedResource(auth *authResult, collectionID, identifier string) (taggedResource, error) {
	token, err := t.loadToken(auth, &wrappers.StringValue{Value: identifier})
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (t *tokenService) CreateToken(ctx context.Context, req *apipb.Token) (*apipb.Token, error) {
	auth, err := t.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "Must specify token")
	}
	if req.Resource == nil || strings.TrimSpace(req.Resource.Value) == "" {
		return nil, status.Error(codes.InvalidArgument, "Resource must be specified")
	}
	if req.Write == nil {
		return nil, status.Error(codes.InvalidArgument, "Write must be specified")
	}
	newToken := model.NewToken()
	newToken.Resource = req.Resource.Value
	newToken.Write = req.Write.Value
	newToken.UserID = auth.User.ID

	if req.Tags != nil {
		for k, v := range req.Tags {
			if !newToken.Tags.IsValidTag(k, v) {
				return nil, status.Error(codes.InvalidArgument, "Invalid key/value for tag")
			}
			newToken.Tags.SetTag(k, v)
		}
	}
	if err := newToken.GenerateToken(); err != nil {
		logging.Warning("Unable to generate token for user %d: %v", newToken.UserID, err)
		return nil, status.Error(codes.Internal, "Could not generate token")
	}

	if err := t.store.CreateToken(newToken); err != nil {
		logging.Warning("Error storing token for user %d: %v", newToken.UserID, err)
		return nil, status.Error(codes.Internal, "Error creating token")
	}
	return apitoolbox.NewTokenFromModel(newToken), nil
}

func (t *tokenService) ListTokens(ctx context.Context, req *apipb.ListTokenRequest) (*apipb.TokenList, error) {
	auth, err := t.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	list, err := t.store.ListTokens(auth.User.ID)
	if err != nil {
		logging.Warning("Error doing token list lookup for user %d: %v", auth.User.ID, err)
		return nil, status.Error(codes.Internal, "Unable to list tokens")
	}
	ret := &apipb.TokenList{
		Tokens: make([]*apipb.Token, 0),
	}
	for _, v := range list {
		ret.Tokens = append(ret.Tokens, apitoolbox.NewTokenFromModel(v))
	}
	return ret, nil
}

func (t *tokenService) DeleteToken(ctx context.Context, req *apipb.DeleteTokenRequest) (*apipb.DeleteTokenResponse, error) {
	auth, err := t.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil || req.Token == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing token")
	}

	if err := t.store.DeleteToken(auth.User.ID, req.Token.Value); err != nil {
		if err == storage.ErrNotFound {
			return nil, status.Error(codes.NotFound, "Unknown token")
		}
		return nil, status.Error(codes.Internal, "Could not delete token")
	}
	return &apipb.DeleteTokenResponse{}, nil
}

func (t *tokenService) RetrieveToken(ctx context.Context, req *apipb.TokenRequest) (*apipb.Token, error) {
	auth, err := t.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing token")
	}
	token, err := t.loadToken(auth, req.Token)
	if err != nil {
		return nil, err
	}
	return apitoolbox.NewTokenFromModel(token), nil
}

func (t *tokenService) UpdateToken(ctx context.Context, req *apipb.Token) (*apipb.Token, error) {
	auth, err := t.EnsureAuth(ctx)
	if err != nil {
		return nil, err
	}

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing request object")
	}

	token, err := t.loadToken(auth, req.Token)
	if err != nil {
		return nil, err
	}

	if req.Resource != nil {
		if strings.TrimSpace(req.Resource.Value) == "" {
			return nil, status.Error(codes.InvalidArgument, "Invalid resource")
		}
		token.Resource = req.Resource.Value
	}
	if req.Write != nil {
		token.Write = req.Write.Value
	}
	if req.Tags != nil {
		for k, v := range req.Tags {
			if !token.Tags.IsValidTag(k, v) {
				return nil, status.Error(codes.InvalidArgument, "Invalid key/value for tag")
			}
			token.SetTag(k, v)
		}
	}
	if err := t.store.UpdateToken(token); err != nil {
		logging.Warning("Error updating token: %v", err)
		return nil, status.Error(codes.Internal, "Unable to update token")
	}
	return apitoolbox.NewTokenFromModel(token), nil
}

func (t *tokenService) ListTokenTags(ctx context.Context, req *apipb.TagRequest) (*apipb.TagResponse, error) {
	return listTags(ctx, req, t)
}

func (t *tokenService) UpdateResourceTags(userID model.UserKey, collectionID, identifier string, res interface{}) error {
	token := res.(*model.Token)
	return t.store.UpdateTokenTags(userID, identifier, token.Tags)
}

func (t *tokenService) UpdateTokenTags(ctx context.Context, req *apipb.UpdateTagRequest) (*apipb.TagResponse, error) {
	return updateTags(ctx, req, t)
}

func (t *tokenService) GetTokenTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return getTag(ctx, req, t)
}

func (t *tokenService) DeleteTokenTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return deleteTag(ctx, req, t)
}

func (t *tokenService) UpdateTokenTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return updateTag(ctx, req, t)
}
