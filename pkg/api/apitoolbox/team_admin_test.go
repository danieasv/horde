package apitoolbox

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
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTestAdmin(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	assert.NotNil(store)
	// Create two users and two teams

	t1 := model.NewTeam()
	t1.ID = store.NewTeamID()
	userID := store.NewUserID()
	u1 := model.NewUser(userID, userID.String(), model.AuthConnectID, t1.ID)
	assert.NoError(store.CreateUser(u1, t1))

	t2 := model.NewTeam()
	t2.ID = store.NewTeamID()
	userID = store.NewUserID()
	u2 := model.NewUser(userID, userID.String(), model.AuthGitHub, t2.ID)
	assert.NoError(store.CreateUser(u2, t2))

	team := model.NewTeam()
	team.ID = store.NewTeamID()
	team.AddMember(model.NewMember(u1, model.AdminRole))
	team.AddMember(model.NewMember(u2, model.MemberRole))
	assert.NoError(store.CreateTeam(team))

	tt, err := EnsureTeamAdmin(u1.ID, team.ID.String(), store)
	assert.NoError(err)
	assert.Equal(team.ID, tt.ID)

	_, err = EnsureTeamAdmin(u2.ID, team.ID.String(), store)
	assert.Error(err)
	assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

	_, err = EnsureTeamAdmin(u2.ID, t1.ID.String(), store)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	_, err = EnsureTeamAdmin(u1.ID, "foo bar baz", store)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
}
