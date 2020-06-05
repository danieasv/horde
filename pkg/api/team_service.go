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
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type teamService struct {
	store storage.DataStore
	defaultGrpcAuth
}

// newTeamService creates a new gRPC team service. The team service handles
// team management and team memberships. Teams are the only entity that can
// have ownership to resources in Horde. Teams are built by the users via
// invites; a team adminstrator (ie user that owns the team) generates an
// invite and sends the invite to another user. When the invite is accepted
// the user is added to the team. Team administrators can set the role for
// other users, ie assign an administrator role to another user. Every team
// must have one or more administrators. You are not allowed to change your
// own role in a team so we end up with always having at least one administrator.
// When a team is created the user that created the team will be the administrator
// of that team.
func newTeamService(store storage.DataStore) teamService {
	return teamService{
		store:           store,
		defaultGrpcAuth: defaultGrpcAuth{Store: store},
	}
}

func (s *teamService) CreateTeam(ctx context.Context, req *apipb.Team) (*apipb.Team, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "Missing team parameter")
	}
	auth := gRPCAuth(ctx, s.store)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "Must authenticate")
	}
	newTeam := model.NewTeam()
	newTeam.ID = s.store.NewTeamID()
	for k, v := range req.Tags {
		if !newTeam.Tags.IsValidTag(k, v) {
			return nil, status.Error(codes.InvalidArgument, "Invalid tag name/value")
		}
		newTeam.Tags.SetTag(k, v)
	}
	// Now that the tags are OK we can create a new team ID
	newTeam.AddMember(model.NewMember(auth.User, model.AdminRole))
	if err := s.store.CreateTeam(newTeam); err != nil {
		logging.Warning("Could not create team: %v", err)
		return nil, status.Error(codes.Internal, "Unable to create team")
	}

	return apitoolbox.NewTeamFromModel(newTeam, true), nil
}

func (s *teamService) LoadTaggedResource(auth *authResult, collectionID, identifier string) (taggedResource, error) {
	team, err := s.loadTeam(auth, identifier)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (s *teamService) UpdateResourceTags(id model.UserKey, collectionID, identifier string, res interface{}) error {
	team := res.(*model.Team)
	return s.store.UpdateTeamTags(id, identifier, team.Tags)
}

func (s *teamService) loadTeam(auth *authResult, identifier string) (model.Team, error) {
	teamID, err := model.NewTeamKeyFromString(identifier)
	if err != nil {
		return model.Team{}, status.Error(codes.InvalidArgument, "Invalid team ID")
	}

	team, err := s.store.RetrieveTeam(auth.User.ID, teamID)
	if err != nil {
		if err == storage.ErrNotFound {
			return model.Team{}, status.Error(codes.NotFound, "Unknown team")
		}
		logging.Warning("Error retrieving team %s: %v", teamID, err)
		return model.Team{}, status.Error(codes.Internal, "Unable to retrieve team")
	}
	return team, nil
}

func (s *teamService) authAndLoadTeam(ctx context.Context, t *wrappers.StringValue) (*authResult, model.Team, error) {
	auth, err := s.EnsureAuth(ctx)
	if err != nil {
		return nil, model.Team{}, err
	}
	if t == nil {
		return auth, model.Team{}, status.Error(codes.InvalidArgument, "Missing team ID")
	}
	team, err := s.loadTeam(auth, t.Value)
	return auth, team, err
}

func (s *teamService) RetrieveTeam(ctx context.Context, req *apipb.TeamRequest) (*apipb.Team, error) {
	if req == nil || req.TeamId == nil {
		return nil, status.Error(codes.InvalidArgument, "No request object")
	}
	_, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}
	return apitoolbox.NewTeamFromModel(team, true), nil
}

func (s *teamService) RetrieveTeamMembers(ctx context.Context, req *apipb.TeamRequest) (*apipb.MemberList, error) {
	if req == nil || req.TeamId == nil {
		return nil, status.Error(codes.InvalidArgument, "No request object")
	}
	_, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}

	ret := &apipb.MemberList{
		Members: make([]*apipb.Member, 0),
	}

	for _, v := range team.Members {
		ret.Members = append(ret.Members, apitoolbox.NewMemberFromModel(team.ID, v))
	}
	return ret, nil
}

func (s *teamService) RetrieveMember(ctx context.Context, req *apipb.MemberRequest) (*apipb.Member, error) {
	if req == nil || req.TeamId == nil || req.UserId == nil {
		return nil, status.Error(codes.InvalidArgument, "No request object")
	}
	_, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}
	userID, err := model.NewUserKeyFromString(req.UserId.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid user ID")
	}

	for _, v := range team.Members {
		if v.User.ID == userID {
			return apitoolbox.NewMemberFromModel(team.ID, v), nil
		}
	}
	return nil, status.Error(codes.NotFound, "Unknown member")
}

func (s *teamService) UpdateMember(ctx context.Context, req *apipb.Member) (*apipb.Member, error) {
	if req == nil || req.TeamId == nil || req.UserId == nil || req.Role == nil {
		return nil, status.Error(codes.InvalidArgument, "Needs user and role to update member")
	}
	auth, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}
	userID, err := model.NewUserKeyFromString(req.UserId.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid user ID")
	}
	roleID := model.NewRoleIDFromString(req.Role.Value)

	if userID == auth.User.ID {
		return nil, status.Error(codes.PermissionDenied, "You are not allowed to change your own membershi")
	}

	if !team.IsMember(userID) {
		return nil, status.Error(codes.InvalidArgument, "User is not a member of the team")
	}
	team.UpdateMember(userID, roleID)

	if err := s.store.UpdateTeam(auth.User.ID, team); err != nil {
		logging.Warning("Error updating team %d: %v", team.ID, err)
		return nil, status.Error(codes.Internal, "Unable to update member state")
	}
	return apitoolbox.NewMemberFromModel(team.ID, team.GetMember(userID)), nil
}

func (s *teamService) DeleteMember(ctx context.Context, req *apipb.MemberRequest) (*apipb.Member, error) {
	if req == nil || req.TeamId == nil || req.UserId == nil {
		return nil, status.Error(codes.InvalidArgument, "Needs user and role to update member")
	}
	auth, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}
	userID, err := model.NewUserKeyFromString(req.UserId.Value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid user ID")
	}
	if userID == auth.User.ID {
		return nil, status.Error(codes.PermissionDenied, "You are not allowed to change your own membershi")
	}
	if !team.IsMember(userID) {
		return nil, status.Error(codes.InvalidArgument, "User is not a member of the team")
	}
	member := team.GetMember(userID)
	team.RemoveMember(userID)

	if err := s.store.UpdateTeam(auth.User.ID, team); err != nil {
		logging.Warning("Error updating team %d: %v", team.ID, err)
		return nil, status.Error(codes.Internal, "Unable to remove member")
	}
	return apitoolbox.NewMemberFromModel(team.ID, member), nil
}

func (s *teamService) UpdateTeam(ctx context.Context, req *apipb.Team) (*apipb.Team, error) {
	if req == nil || req.TeamId == nil {
		return nil, status.Error(codes.InvalidArgument, "Needs team ID to update team")
	}
	auth, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}

	// Team update is basically just the tags
	if req.Tags != nil {
		for k, v := range req.Tags {
			if !team.IsValidTag(k, v) {
				return nil, status.Error(codes.InvalidArgument, "Invalid tag name/value")
			}
			team.SetTag(k, v)
		}
	}
	if err := s.store.UpdateTeamTags(auth.User.ID, team.ID.String(), team.Tags); err != nil {
		if err == storage.ErrAccess {
			// User isn't an admin
			return nil, status.Error(codes.PermissionDenied, "Must be administrator to update team")
		}
		logging.Warning("Unable to update team %d: %v", team.ID, err)
		return nil, status.Error(codes.Internal, "Unable to update team")
	}
	return apitoolbox.NewTeamFromModel(team, true), nil
}

func (s *teamService) DeleteTeam(ctx context.Context, req *apipb.TeamRequest) (*apipb.Team, error) {
	if req == nil || req.TeamId == nil {
		return nil, status.Error(codes.InvalidArgument, "Needs team ID to update team")
	}
	auth, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}

	if team.ID == auth.User.PrivateTeamID {
		return nil, status.Error(codes.PermissionDenied, "You can't remove your private team")
	}

	if err := s.store.DeleteTeam(auth.User.ID, team.ID); err != nil {
		return nil, status.Error(codes.Internal, "Unable to remove team")
	}

	return apitoolbox.NewTeamFromModel(team, true), nil
}

func (s *teamService) ListTeams(ctx context.Context, req *apipb.ListTeamRequest) (*apipb.TeamList, error) {
	auth := gRPCAuth(ctx, s.store)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "Must authenticate")
	}

	teams, err := s.store.ListTeams(auth.User.ID)
	if err != nil {
		logging.Warning("Unable to list teams: %v (user ID=%d)", err, auth.User.ID)
		return nil, status.Error(codes.Internal, "Unable to list teams")
	}

	ret := &apipb.TeamList{
		Teams: make([]*apipb.Team, 0),
	}

	for _, v := range teams {
		ret.Teams = append(ret.Teams, apitoolbox.NewTeamFromModel(v, true))
	}
	return ret, nil
}

func (s *teamService) GenerateInvite(ctx context.Context, req *apipb.InviteRequest) (*apipb.Invite, error) {
	if req == nil || req.TeamId == nil {
		return nil, status.Error(codes.InvalidArgument, "Must specify team ID")
	}

	auth, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}

	if auth.User.PrivateTeamID == team.ID {
		return nil, status.Error(codes.InvalidArgument, "Can't invite people to private team")
	}

	invite, err := model.NewInvite(auth.User.ID, team.ID)
	if err != nil {
		logging.Warning("Unable to create invite for team %d (user id=%d): %v", team.ID, auth.User.ID, err)
		return nil, status.Error(codes.Internal, "Unable to create an invite")
	}
	if err := s.store.CreateInvite(invite); err != nil {
		return nil, status.Error(codes.Internal, "Unable to store created invite")
	}
	return apitoolbox.NewInviteFromModel(invite), nil
}

func (s *teamService) AcceptInvite(ctx context.Context, req *apipb.AcceptInviteRequest) (*apipb.Team, error) {
	if req == nil || req.Code == nil {
		return nil, status.Error(codes.InvalidArgument, "Must specify invite code")
	}
	auth := gRPCAuth(ctx, s.store)
	if auth == nil {
		return nil, status.Error(codes.Unauthenticated, "Must authenticate")
	}
	invite, err := s.store.RetrieveInvite(req.Code.Value)
	if err != nil {
		if err == storage.ErrNotFound {
			return nil, status.Error(codes.NotFound, "Unknown invite code. The invite might have been used.")
		}
		logging.Warning("Unable to retrieve invite %s: %v", req.Code.Value, err)
		return nil, status.Error(codes.Internal, "Error retrieving invite")
	}

	if err := s.store.AcceptInvite(invite, auth.User.ID); err != nil {
		return nil, status.Error(codes.Internal, "Could not accept invite")
	}

	logging.Debug("Invite with code %s generated by %d accepted %d", invite.Code, invite.UserID, auth.User.ID)
	team, err := s.store.RetrieveTeam(auth.User.ID, invite.TeamID)
	if err != nil {
		logging.Warning("Unable to retrieve team %d: %v", invite.TeamID, err)
		return nil, status.Error(codes.Internal, "Error retrieving team for invite")
	}
	return apitoolbox.NewTeamFromModel(team, true), nil
}

func (s *teamService) ListInvites(ctx context.Context, req *apipb.TeamRequest) (*apipb.InviteList, error) {
	if req == nil || req.TeamId == nil {
		return nil, status.Error(codes.InvalidArgument, "Must specify team ID")
	}

	auth, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}

	if !team.IsAdmin(auth.User.ID) {
		return nil, status.Error(codes.PermissionDenied, "Must be administrator to list invites")
	}

	invites, err := s.store.ListInvites(team.ID, auth.User.ID)
	if err != nil {
		logging.Warning("Error loading invite list for team %d: %v", team.ID, err)
		return nil, status.Error(codes.Internal, "Error loading invite list")
	}

	ret := &apipb.InviteList{
		Invites: make([]*apipb.Invite, 0),
	}
	for _, v := range invites {
		ret.Invites = append(ret.Invites, apitoolbox.NewInviteFromModel(v))
	}
	return ret, nil
}

func (s *teamService) RetrieveInvite(ctx context.Context, req *apipb.InviteRequest) (*apipb.Invite, error) {
	if req == nil || req.TeamId == nil || req.Code == nil {
		return nil, status.Error(codes.InvalidArgument, "Must specify team ID and code")
	}

	auth, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}
	if !team.IsAdmin(auth.User.ID) {
		return nil, status.Error(codes.PermissionDenied, "Must be administrator to see invites")
	}

	invites, err := s.store.ListInvites(team.ID, auth.User.ID)
	if err != nil {
		logging.Warning("Error loading invite list for team %d: %v", team.ID, err)
		return nil, status.Error(codes.Internal, "Error loading invite list")
	}

	for _, v := range invites {
		if req.Code.Value == v.Code {
			return apitoolbox.NewInviteFromModel(v), nil
		}
	}
	return nil, status.Error(codes.NotFound, "Unknown invite")
}

func (s *teamService) DeleteInvite(ctx context.Context, req *apipb.InviteRequest) (*apipb.DeleteInviteResponse, error) {
	if req == nil || req.TeamId == nil || req.Code == nil {
		return nil, status.Error(codes.InvalidArgument, "Must specify team ID and code")
	}

	auth, team, err := s.authAndLoadTeam(ctx, req.TeamId)
	if err != nil {
		return nil, err
	}
	if !team.IsAdmin(auth.User.ID) {
		return nil, status.Error(codes.PermissionDenied, "Must be administrator to see invites")
	}

	// Return error unless there's no error or ErrNotFound is returned.
	// Not found is OK since the request has succeeded. It's very REST-like
	// behaviour
	if err := s.store.DeleteInvite(req.Code.Value, team.ID, auth.User.ID); err != nil && err != storage.ErrNotFound {
		logging.Warning("Error loading invite list for team %d: %v", team.ID, err)
		return nil, status.Error(codes.Internal, "Error removing invite")
	}
	return &apipb.DeleteInviteResponse{}, nil
}

func (s *teamService) ListTeamTags(ctx context.Context, req *apipb.TagRequest) (*apipb.TagResponse, error) {
	return listTags(ctx, req, s)
}

func (s *teamService) UpdateTeamTags(ctx context.Context, req *apipb.UpdateTagRequest) (*apipb.TagResponse, error) {
	return updateTags(ctx, req, s)
}

func (s *teamService) GetTeamTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return getTag(ctx, req, s)
}

func (s *teamService) DeleteTeamTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return deleteTag(ctx, req, s)
}

func (s *teamService) UpdateTeamTag(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error) {
	return updateTag(ctx, req, s)
}
