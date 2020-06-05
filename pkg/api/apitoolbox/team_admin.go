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
	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EnsureTeamAdmin ensures that the user is an admin of the team. The team
// will be retrieved from the storage implementation supplied. If the team
// ID is unknown a suitable gRPC error with an error code will be returned
// (InvalidArgument if the ID can't be converted into a model.TeamKey value,
// NotFound if the team doesn't exist or PermissionDenied if the user isn't an
// admin of the team)
func EnsureTeamAdmin(userID model.UserKey, id string, store storage.DataStore) (model.Team, error) {
	// Team ID is included. Ensure it is valid and that the logged in user
	// is an administrator of that team.
	teamID, err := model.NewTeamKeyFromString(id)
	if err != nil {
		return model.Team{}, status.Error(codes.InvalidArgument, "Team ID is invalid")
	}
	team, err := store.RetrieveTeam(userID, teamID)
	if err != nil {
		if err == storage.ErrNotFound {
			return model.Team{}, status.Error(codes.NotFound, "Unknown team ID")
		}
		logging.Warning("Error retrieving team %d: %v", teamID, err)
		return model.Team{}, status.Error(codes.Internal, "Unable to read team")
	}
	if !team.IsAdmin(userID) {
		return model.Team{}, status.Error(codes.PermissionDenied, "Must be administrator of team")
	}

	return team, nil
}
