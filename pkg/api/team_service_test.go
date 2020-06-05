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

	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type teamRequestFactory struct {
}

func (f *teamRequestFactory) ValidRequest() interface{} {
	return &apipb.TeamRequest{}
}

func (f *teamRequestFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.TeamRequest).TeamId = cid
}

func (f *teamRequestFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	// Nothing
}

func TestCreateUpdateDeleteTeam(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	teamService := newTeamService(store)
	assert.NotNil(teamService)
	user, _, ctx := createAuthenticatedContext(assert, store)

	t1 := model.NewTeam()
	t1.ID = store.NewTeamID()
	t1.AddMember(model.NewMember(user, model.AdminRole))
	assert.NoError(store.CreateTeam(t1))

	genericRequestTests(tparam{
		AuthenticatedContext:      ctx,
		Assert:                    assert,
		CollectionID:              t1.ID.String(),
		IdentifierID:              "",
		TestWithInvalidIdentifier: false,
		RequestFactory:            &teamRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return teamService.DeleteTeam(ctx, nil)
			}
			return teamService.DeleteTeam(ctx, req.(*apipb.TeamRequest))
		}})

	// Ensure we authenticate
	_, err := teamService.CreateTeam(context.Background(), &apipb.Team{})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	_, err = teamService.UpdateTeam(context.Background(), &apipb.Team{
		TeamId: &wrappers.StringValue{Value: "123"},
	})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Use nil parameter. Should fail
	_, err = teamService.CreateTeam(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = teamService.UpdateTeam(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Create a team. This should work and the team should be stored in the
	// backend store
	team := &apipb.Team{Tags: map[string]string{
		"foo": "bar",
		"bar": "baz",
		"baz": "foo",
	}}
	team, err = teamService.CreateTeam(ctx, team)
	assert.NoError(err)
	assert.NotNil(team)

	teamKey, err := model.NewTeamKeyFromString(team.TeamId.Value)
	assert.NoError(err)

	storedTeam, err := store.RetrieveTeam(user.ID, teamKey)
	assert.NoError(err)
	assert.Equal(storedTeam.ID.String(), team.TeamId.Value)
	assert.Equal("bar", storedTeam.GetTag("foo"))
	assert.Equal("baz", storedTeam.GetTag("bar"))
	assert.Equal("foo", storedTeam.GetTag("baz"))

	// Update the team with new tags
	team.Tags["foobar"] = "barbaz"
	t.Logf("Team: %+v", team)
	t2, err := teamService.UpdateTeam(ctx, team)
	assert.NoError(err)
	assert.Contains(t2.Tags, "foobar")

	// use invalid team tags. Should fail
	t2.Tags[""] = "Invalid name"
	_, err = teamService.UpdateTeam(ctx, t2)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = teamService.CreateTeam(ctx, &apipb.Team{Tags: map[string]string{
		"": "bar",
	}})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Deleting private team should fail
	_, err = teamService.DeleteTeam(ctx, &apipb.TeamRequest{
		TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()},
	})
	assert.Error(err)
	assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

	// Deleting a non-private team is OK
	t2, err = teamService.DeleteTeam(ctx, &apipb.TeamRequest{TeamId: t2.TeamId})
	assert.NoError(err)
	assert.NotNil(t2)
}

func TestListRetrieveTeam(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	teamService := newTeamService(store)
	assert.NotNil(teamService)
	user, _, ctx := createAuthenticatedContext(assert, store)

	genericRequestTests(tparam{
		Assert:                    assert,
		AuthenticatedContext:      ctx,
		CollectionID:              user.PrivateTeamID.String(),
		IdentifierID:              "",
		TestWithInvalidIdentifier: false,
		RequestFactory:            &teamRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return teamService.RetrieveTeam(ctx, nil)
			}
			return teamService.RetrieveTeam(ctx, req.(*apipb.TeamRequest))
		}})

	// Ensure we authenticate the requests
	_, err := teamService.ListTeams(context.Background(), &apipb.ListTeamRequest{})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Look up the private team ID. Should be returned.
	list, err := teamService.ListTeams(ctx, &apipb.ListTeamRequest{})
	assert.NoError(err)
	assert.NotNil(list)
	assert.Len(list.Teams, 1)

	// Use nil value for request, then for team id
	_, err = teamService.RetrieveTeam(ctx, &apipb.TeamRequest{TeamId: nil})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
}

func TestRetrieveTeamMembers(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	teamService := newTeamService(store)
	assert.NotNil(teamService)

	// Ensure we authenticate the request
	res, err := teamService.RetrieveTeamMembers(context.Background(), &apipb.TeamRequest{TeamId: &wrappers.StringValue{Value: "123"}})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	user, _, ctx := createAuthenticatedContext(assert, store)

	// Look up the private team ID. Should be returned.
	res, err = teamService.RetrieveTeamMembers(ctx, &apipb.TeamRequest{TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()}})
	assert.NoError(err)
	assert.NotNil(res)
	assert.Len(res.Members, 1)
	assert.Equal(user.ID.String(), res.Members[0].UserId.Value)
	assert.Equal(user.PrivateTeamID.String(), res.Members[0].TeamId.Value)

	// Use nil team ID and nil request object
	_, err = teamService.RetrieveTeamMembers(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = teamService.RetrieveTeamMembers(ctx, &apipb.TeamRequest{TeamId: nil})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Use invalid team ID for request. Should return InvalidArgument
	res, err = teamService.RetrieveTeamMembers(ctx, &apipb.TeamRequest{TeamId: &wrappers.StringValue{Value: "12x"}})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Use valid but unknown team ID. Should return NotFound
	res, err = teamService.RetrieveTeamMembers(ctx, &apipb.TeamRequest{TeamId: &wrappers.StringValue{Value: "0"}})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())
}

func TestRetrieveMember(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	teamService := newTeamService(store)
	assert.NotNil(teamService)

	// Ensure we authenticate the request
	res, err := teamService.RetrieveMember(context.Background(), &apipb.MemberRequest{
		TeamId: &wrappers.StringValue{Value: "123"},
		UserId: &wrappers.StringValue{Value: "123"}})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	user, _, ctx := createAuthenticatedContext(assert, store)

	user2, _ := createUserAndToken(assert, model.AuthInternal, store)
	team := model.NewTeam()
	team.ID = store.NewTeamID()
	team.AddMember(model.NewMember(user, model.AdminRole))
	team.AddMember(model.NewMember(user2, model.MemberRole))
	assert.NoError(store.CreateTeam(team))

	// Retrieve the member
	res, err = teamService.RetrieveMember(ctx, &apipb.MemberRequest{
		UserId: &wrappers.StringValue{Value: user2.ID.String()},
		TeamId: &wrappers.StringValue{Value: team.ID.String()},
	})
	assert.NoError(err)
	assert.NotNil(res)

	// Retrieve a member that doesn't exist in the team. Use the private team
	// for user #1
	res, err = teamService.RetrieveMember(ctx, &apipb.MemberRequest{
		UserId: &wrappers.StringValue{Value: user2.ID.String()},
		TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Test nil user id, team id and request
	_, err = teamService.RetrieveMember(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	_, err = teamService.RetrieveMember(ctx, &apipb.MemberRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Use invalid team ID and user ID string
	_, err = teamService.RetrieveMember(ctx, &apipb.MemberRequest{
		TeamId: &wrappers.StringValue{Value: "xxy"},
		UserId: &wrappers.StringValue{Value: "123"},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = teamService.RetrieveMember(ctx, &apipb.MemberRequest{
		TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()},
		UserId: &wrappers.StringValue{Value: "xxy"},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Use unknown team ID
	_, err = teamService.RetrieveMember(ctx, &apipb.MemberRequest{
		TeamId: &wrappers.StringValue{Value: "0"},
		UserId: &wrappers.StringValue{Value: "123"},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

}

func TestUpdateDeleteMember(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	teamService := newTeamService(store)
	assert.NotNil(teamService)

	// Ensure we authenticate the requests
	res, err := teamService.UpdateMember(context.Background(), &apipb.Member{
		TeamId: &wrappers.StringValue{Value: "123"},
		UserId: &wrappers.StringValue{Value: "123"},
		Role:   &wrappers.StringValue{Value: "admin"},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	res, err = teamService.DeleteMember(context.Background(), &apipb.MemberRequest{
		TeamId: &wrappers.StringValue{Value: "123"},
		UserId: &wrappers.StringValue{Value: "123"},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	user, _, ctx := createAuthenticatedContext(assert, store)

	// Use nil requests
	res, err = teamService.UpdateMember(ctx, nil)
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	res, err = teamService.DeleteMember(ctx, nil)
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Use invalid user key values for request
	res, err = teamService.UpdateMember(ctx, &apipb.Member{
		TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()},
		UserId: &wrappers.StringValue{Value: "xxx"},
		Role:   &wrappers.StringValue{Value: "admin"},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	res, err = teamService.DeleteMember(ctx, &apipb.MemberRequest{
		TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()},
		UserId: &wrappers.StringValue{Value: "xxx"},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Ensure we can't delete or modify ourselves
	res, err = teamService.UpdateMember(ctx, &apipb.Member{
		TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()},
		UserId: &wrappers.StringValue{Value: user.ID.String()},
		Role:   &wrappers.StringValue{Value: "member"},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

	res, err = teamService.DeleteMember(ctx, &apipb.MemberRequest{
		TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()},
		UserId: &wrappers.StringValue{Value: user.ID.String()},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

	// Ensure we get an error if we try to modify a non-member
	user2, _ := createUserAndToken(assert, model.AuthInternal, store)
	res, err = teamService.UpdateMember(ctx, &apipb.Member{
		TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()},
		UserId: &wrappers.StringValue{Value: user2.ID.String()},
		Role:   &wrappers.StringValue{Value: "member"},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	res, err = teamService.DeleteMember(ctx, &apipb.MemberRequest{
		TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()},
		UserId: &wrappers.StringValue{Value: user2.ID.String()},
	})
	assert.Error(err)
	assert.Nil(res)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Create a new team with both users as members of the team
	team := model.NewTeam()
	team.ID = store.NewTeamID()
	team.AddMember(model.NewMember(user, model.AdminRole))
	team.AddMember(model.NewMember(user2, model.MemberRole))
	assert.NoError(store.CreateTeam(team))

	// It should be possible to change the 2nd user into an admin
	res, err = teamService.UpdateMember(ctx, &apipb.Member{
		TeamId: &wrappers.StringValue{Value: team.ID.String()},
		UserId: &wrappers.StringValue{Value: user2.ID.String()},
		Role:   &wrappers.StringValue{Value: "admin"},
	})
	assert.NoError(err)
	assert.Equal(user2.ID.String(), res.UserId.Value)
	assert.Equal(team.ID.String(), res.TeamId.Value)
	assert.Equal(model.AdminRole.String(), res.Role.Value)

	// ...and to remove the 2nd user
	res, err = teamService.DeleteMember(ctx, &apipb.MemberRequest{
		TeamId: &wrappers.StringValue{Value: team.ID.String()},
		UserId: &wrappers.StringValue{Value: user2.ID.String()},
	})
	assert.NoError(err)
	assert.Equal(user2.ID.String(), res.UserId.Value)
	assert.Equal(team.ID.String(), res.TeamId.Value)
	assert.Equal(model.AdminRole.String(), res.Role.Value)
}

type inviteRequestFactory struct {
}

func (f *inviteRequestFactory) ValidRequest() interface{} {
	return &apipb.InviteRequest{}
}

func (f *inviteRequestFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.InviteRequest).TeamId = cid
}

func (f *inviteRequestFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	// Nothing
}
func TestInvites(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	teamService := newTeamService(store)
	assert.NotNil(teamService)

	user, _, ctx := createAuthenticatedContext(assert, store)

	t0 := model.NewTeam()
	t0.ID = store.NewTeamID()
	t0.AddMember(model.NewMember(user, model.AdminRole))
	assert.NoError(store.CreateTeam(t0))

	genericRequestTests(tparam{
		Assert:                    assert,
		AuthenticatedContext:      ctx,
		CollectionID:              user.PrivateTeamID.String(),
		IdentifierID:              "",
		TestWithInvalidIdentifier: false,
		RequestFactory:            &teamRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return teamService.ListInvites(ctx, nil)
			}
			return teamService.ListInvites(ctx, req.(*apipb.TeamRequest))
		}})

	team := model.NewTeam()
	team.ID = store.NewTeamID()
	team.AddMember(model.NewMember(user, model.AdminRole))
	assert.NoError(store.CreateTeam(team))

	genericRequestTests(tparam{
		Assert:                    assert,
		AuthenticatedContext:      ctx,
		CollectionID:              t0.ID.String(),
		IdentifierID:              "",
		TestWithInvalidIdentifier: false,
		RequestFactory:            &inviteRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return teamService.GenerateInvite(ctx, nil)
			}
			return teamService.GenerateInvite(ctx, req.(*apipb.InviteRequest))
		}})

	// Ensure we are authenticated for all the calls

	_, err := teamService.AcceptInvite(context.Background(), &apipb.AcceptInviteRequest{
		Code: &wrappers.StringValue{Value: "foof"},
	})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	_, err = teamService.RetrieveInvite(context.Background(), &apipb.InviteRequest{
		TeamId: &wrappers.StringValue{Value: "123"},
		Code:   &wrappers.StringValue{Value: "code"},
	})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	_, err = teamService.DeleteInvite(context.Background(), &apipb.InviteRequest{
		TeamId: &wrappers.StringValue{Value: "123"},
		Code:   &wrappers.StringValue{Value: "code"},
	})
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Nil requests returns invalid
	_, err = teamService.AcceptInvite(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	_, err = teamService.AcceptInvite(ctx, &apipb.AcceptInviteRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = teamService.RetrieveInvite(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	_, err = teamService.RetrieveInvite(ctx, &apipb.InviteRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	_, err = teamService.RetrieveInvite(ctx, &apipb.InviteRequest{
		TeamId: &wrappers.StringValue{Value: "123"},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = teamService.DeleteInvite(ctx, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	_, err = teamService.DeleteInvite(ctx, &apipb.InviteRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	_, err = teamService.DeleteInvite(ctx, &apipb.InviteRequest{
		TeamId: &wrappers.StringValue{Value: "123"},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Can't generate invite for private team (it's private after all)
	_, err = teamService.GenerateInvite(ctx, &apipb.InviteRequest{
		TeamId: &wrappers.StringValue{Value: user.PrivateTeamID.String()},
	})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Create a team (in storage) that we can use

	// Generate the invite
	invite, err := teamService.GenerateInvite(ctx, &apipb.InviteRequest{
		TeamId: &wrappers.StringValue{Value: team.ID.String()},
	})
	assert.NoError(err)
	assert.NotNil(invite)

	// List invites on the invite resource
	_, err = teamService.ListInvites(ctx, &apipb.TeamRequest{
		TeamId: &wrappers.StringValue{Value: team.ID.String()},
	})
	assert.NoError(err)

	// Create a 2nd user to accept the invite
	_, _, ctx2 := createAuthenticatedContext(assert, store)

	res, err := teamService.AcceptInvite(ctx2, &apipb.AcceptInviteRequest{
		Code: invite.Code,
	})
	assert.NoError(err)
	assert.NotNil(res)

	// Accepting the invite a 2nd time will fail with NotFound
	_, err = teamService.AcceptInvite(ctx2, &apipb.AcceptInviteRequest{
		Code: invite.Code,
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	teamRequest := &apipb.TeamRequest{
		TeamId: &wrappers.StringValue{Value: team.ID.String()},
	}
	// The invite list should be empty
	l, err := teamService.ListInvites(ctx, teamRequest)
	assert.NoError(err)
	assert.Len(l.Invites, 0)

	// Create a new invite by the first user
	invite, err = teamService.GenerateInvite(ctx, &apipb.InviteRequest{
		TeamId: &wrappers.StringValue{Value: team.ID.String()},
	})
	assert.NoError(err)
	assert.NotNil(invite)

	// 2nd user should not be able to see invites
	_, err = teamService.ListInvites(ctx2, teamRequest)
	assert.Error(err)
	assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

	_, err = teamService.RetrieveInvite(ctx2, &apipb.InviteRequest{
		TeamId: teamRequest.TeamId,
		Code:   invite.Code,
	})
	assert.Error(err)
	assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

	// The 1st user should be able to read the invite
	inviteRequest := &apipb.InviteRequest{
		TeamId: teamRequest.TeamId,
		Code:   invite.Code,
	}
	i, err := teamService.RetrieveInvite(ctx, inviteRequest)
	assert.NoError(err)
	assert.Equal(i.Code.Value, invite.Code.Value)

	// ...but not unknown invites
	_, err = teamService.RetrieveInvite(ctx, &apipb.InviteRequest{
		TeamId: teamRequest.TeamId,
		Code:   &wrappers.StringValue{Value: "unknown"},
	})
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// 2nd user should not be able to delete it
	_, err = teamService.DeleteInvite(ctx2, inviteRequest)
	assert.Error(err)
	assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

	// 1st user should be able to delete it
	_, err = teamService.DeleteInvite(ctx, inviteRequest)
	assert.NoError(err)

	// ...twice!
	_, err = teamService.DeleteInvite(ctx, inviteRequest)
	assert.NoError(err)

}

func TestTeamTags(t *testing.T) {
	assert := require.New(t)
	store := sqlstore.NewMemoryStore()
	teamService := newTeamService(store)
	assert.NotNil(teamService)

	user, _, ctx := createAuthenticatedContext(assert, store)

	team := model.NewTeam()
	team.ID = store.NewTeamID()
	team.AddMember(model.NewMember(user, model.AdminRole))
	assert.NoError(store.CreateTeam(team))

	doTagTests(ctx, assert, "", false, team.ID.String(), tagFunctions{
		ListTags:   teamService.ListTeamTags,
		UpdateTag:  teamService.UpdateTeamTag,
		GetTag:     teamService.GetTeamTag,
		DeleteTag:  teamService.DeleteTeamTag,
		UpdateTags: teamService.UpdateTeamTags,
	})
}
