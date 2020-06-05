package htest

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
	"fmt"
	"math/rand"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// GeneratorSettings holds parameters for the GenerateData function
type GeneratorSettings struct {
	CollectionsPerTeam   int
	DevicesPerCollection int
	TeamsPerUser         int
	MembersPerTeam       int
	TokensPerUser        int
	InvitesPerUser       int
	OutputsPerCollection int
	LogActivity          bool
}

// DefaultGeneratorSettings is the default generator settings to use. These are
// best guesses for the actual distribution of entities. Some might be too
// conservative and some might be too optimistic but if you generate 10K users
// you should get reasonable large data sets. The number of users is the
// deciding factor when generating data.
var DefaultGeneratorSettings = GeneratorSettings{
	CollectionsPerTeam:   5,
	DevicesPerCollection: 10000,
	TeamsPerUser:         5,
	MembersPerTeam:       5,
	TokensPerUser:        3,
	InvitesPerUser:       3,
	OutputsPerCollection: 3,
	LogActivity:          false,
}

func createUsers(count int, settings GeneratorSettings, store storage.DataStore) ([]model.User, error) {
	var ret []model.User

	for i := 0; i < count; i++ {
		newUserTeam := model.Team{
			ID: store.NewTeamID(),
		}
		id := store.NewUserID()
		newUser := model.NewUser(id, id.String(), model.AuthConnectID, newUserTeam.ID)
		newUser.Name = fmt.Sprintf("User %v", id)
		if err := store.CreateUser(newUser, newUserTeam); err != nil {
			fmt.Println("Error creating user: ", err)
			return ret, fmt.Errorf("error creating user: %v", err)
		}
		for n := 0; n < settings.TokensPerUser; n++ {
			token := model.NewToken()
			token.GenerateToken()
			token.UserID = newUser.ID
			token.Resource = "/"
			token.Write = rand.Int() == 0
			if err := store.CreateToken(token); err != nil {
				return ret, fmt.Errorf("error creating token: %v", err)
			}
		}
		if settings.LogActivity {
			logging.Info("Created user %s", newUser.ID.String())
		}
		ret = append(ret, newUser)
	}
	return ret, nil
}

func createTeamsCollectionsDevices(users []model.User, settings GeneratorSettings, store storage.DataStore) error {
	for _, v := range users {
		for n := 0; n < settings.TeamsPerUser; n++ {
			newTeam := model.NewTeam()
			newTeam.ID = store.NewTeamID()
			newTeam.AddMember(model.NewMember(v, model.AdminRole))
			for n := 0; n < settings.MembersPerTeam; n++ {
				newTeam.AddMember(model.NewMember(users[rand.Intn(len(users))], model.MemberRole))
			}
			if err := store.CreateTeam(newTeam); err != nil {
				return fmt.Errorf("error creating team: %v", err)
			}
			if err := createCollections(v.ID, newTeam.ID, settings, store); err != nil {
				return err
			}
		}
	}
	return nil
}

func createCollections(userID model.UserKey, teamID model.TeamKey, settings GeneratorSettings, store storage.DataStore) error {
	for n := 0; n < settings.CollectionsPerTeam; n++ {
		newCollection := model.NewCollection()
		newCollection.ID = store.NewCollectionID()
		newCollection.TeamID = teamID

		if settings.LogActivity {
			logging.Info("Creating collection %s", newCollection.ID.String())
		}
		if err := store.CreateCollection(userID, newCollection); err != nil {
			return fmt.Errorf("error creating collection: %v", err)
		}
		if settings.LogActivity {
			logging.Info("Creating devices for collection %s", newCollection.ID.String())
		}
		if err := createDevices(userID, newCollection.ID, settings, store); err != nil {
			return fmt.Errorf("error creating device for collection: %v", err)
		}
		if settings.LogActivity {
			logging.Info("Creating outputs for collection %s", newCollection.ID.String())
		}
		if err := createOutputs(userID, newCollection.ID, settings, store); err != nil {
			return fmt.Errorf("error creating outputs: %v", err)
		}
	}
	return nil
}

func createDevices(userID model.UserKey, collectionID model.CollectionKey, settings GeneratorSettings, store storage.DataStore) error {
	for n := 0; n < settings.DevicesPerCollection; n++ {
		newDevice := model.NewDevice()
		newDevice.ID = store.NewDeviceID()
		newDevice.IMSI = int64(newDevice.ID)
		newDevice.IMEI = int64(newDevice.ID)
		newDevice.CollectionID = collectionID
		newDevice.Network.AllocatedIP = "10.1.0.1"
		newDevice.Network.AllocatedAt = time.Now()
		newDevice.Network.CellID = 1
		newDevice.Firmware.FirmwareVersion = "1.0"
		if err := store.CreateDevice(userID, newDevice); err != nil {
			return err
		}

	}
	return nil
}

func createOutputs(userID model.UserKey, collectionID model.CollectionKey, settings GeneratorSettings, store storage.DataStore) error {
	for n := 0; n < settings.OutputsPerCollection; n++ {
		output := model.NewOutput()
		output.ID = store.NewOutputID()
		output.CollectionID = collectionID
		output.Enabled = false
		output.Type = "udp"
		output.Config = model.NewOutputConfig()
		output.Config["host"] = "localhost"
		output.Config["port"] = 4711
		if err := store.CreateOutput(userID, output); err != nil {
			return err
		}
	}
	return nil
}

// Generate generates a data set in a backend store. The size of the data set is
// determined by the number of users to generate. Multiply the number of
func Generate(userCount int, settings GeneratorSettings, store storage.DataStore) error {

	userList, err := createUsers(userCount, settings, store)
	if err != nil {
		return err
	}
	if err := createTeamsCollectionsDevices(userList, settings, store); err != nil {
		return err
	}
	return nil
}

// TokenCount returns the number of teams generated based on the number of users
// and GeneratorSettings used
func TokenCount(userCount int, settings GeneratorSettings) int {
	return userCount * settings.TokensPerUser
}

// TeamCount returns the number of teams generated based on the number of users
// and GeneratorSettings used
func TeamCount(userCount int, settings GeneratorSettings) int {
	return userCount * settings.TeamsPerUser
}

// MemberCount returns the number of team members generated based on the
// number of users and GeneratorSettings used
func MemberCount(userCount int, settings GeneratorSettings) int {
	return TeamCount(userCount, settings) * settings.MembersPerTeam
}

// CollectionCount returns the number of collections generated based on the
// number of users and GeneratorSettings used
func CollectionCount(userCount int, settings GeneratorSettings) int {
	return TeamCount(userCount, settings) * settings.CollectionsPerTeam
}

// DeviceCount returns the number of collections given the number of users
func DeviceCount(userCount int, settings GeneratorSettings) int {
	return CollectionCount(userCount, settings) * settings.DevicesPerCollection
}

// OutputCount returns the number of collections given the number of users
func OutputCount(userCount int, settings GeneratorSettings) int {
	return CollectionCount(userCount, settings) * settings.OutputsPerCollection
}
