package sqlstore

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
	"sync"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// NewSQLMutexStore wraps a SQL store instance in mutexes, particularly SQLLite3 needs this
func NewSQLMutexStore(driver, connectionString string, create bool, dataCenterID uint8, workerID uint16) (storage.DataStore, error) {
	s, err := NewSQLStore(driver, connectionString, create, dataCenterID, workerID)
	if err != nil {
		return nil, err
	}
	return &mutexWrapper{src: s, m: &sync.Mutex{}}, nil
}

type mutexWrapper struct {
	src storage.DataStore
	m   *sync.Mutex
}

func (m *mutexWrapper) RetrieveUserByExternalID(id string, auth model.AuthMethod) (*model.User, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveUserByExternalID(id, auth)
}

func (m *mutexWrapper) NewUserID() model.UserKey {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.NewUserID()
}
func (m *mutexWrapper) CreateUser(user model.User, privateTeam model.Team) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.CreateUser(user, privateTeam)
}
func (m *mutexWrapper) UpdateUser(user *model.User) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateUser(user)
}
func (m *mutexWrapper) RetrieveUser(userID model.UserKey) (model.User, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveUser(userID)
}
func (m *mutexWrapper) CreateToken(token model.Token) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.CreateToken(token)
}
func (m *mutexWrapper) ListTokens(userID model.UserKey) ([]model.Token, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.ListTokens(userID)
}
func (m *mutexWrapper) UpdateToken(token model.Token) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateToken(token)
}
func (m *mutexWrapper) DeleteToken(userID model.UserKey, token string) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.DeleteToken(userID, token)
}
func (m *mutexWrapper) RetrieveTokenTags(userID model.UserKey, token string) (model.Tags, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveTokenTags(userID, token)
}
func (m *mutexWrapper) UpdateTokenTags(userID model.UserKey, token string, tags model.Tags) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateTokenTags(userID, token, tags)
}
func (m *mutexWrapper) RetrieveToken(token string) (model.Token, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveToken(token)
}
func (m *mutexWrapper) NewDeviceID() model.DeviceKey {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.NewDeviceID()
}
func (m *mutexWrapper) CreateDevice(userID model.UserKey, newDevice model.Device) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.CreateDevice(userID, newDevice)
}
func (m *mutexWrapper) ListDevices(userID model.UserKey, collectionID model.CollectionKey) ([]model.Device, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.ListDevices(userID, collectionID)
}
func (m *mutexWrapper) RetrieveDevice(userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) (model.Device, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveDevice(userID, collectionID, deviceID)
}
func (m *mutexWrapper) DeleteDevice(userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.DeleteDevice(userID, collectionID, deviceID)
}
func (m *mutexWrapper) UpdateDevice(userID model.UserKey, collectionID model.CollectionKey, device model.Device) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateDevice(userID, collectionID, device)
}
func (m *mutexWrapper) RetrieveDeviceTags(userID model.UserKey, deviceID string) (model.Tags, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveDeviceTags(userID, deviceID)
}
func (m *mutexWrapper) UpdateDeviceTags(userID model.UserKey, deviceID string, tags model.Tags) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateDeviceTags(userID, deviceID, tags)
}
func (m *mutexWrapper) RetrieveDeviceByIMSI(imsi int64) (model.Device, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveDeviceByIMSI(imsi)
}
func (m *mutexWrapper) RetrieveDeviceByMSISDN(msisdn string) (model.Device, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveDeviceByMSISDN(msisdn)
}
func (m *mutexWrapper) UpdateDeviceMetadata(device model.Device) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateDeviceMetadata(device)
}
func (m *mutexWrapper) NewCollectionID() model.CollectionKey {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.NewCollectionID()
}
func (m *mutexWrapper) ListCollections(userID model.UserKey) ([]model.Collection, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.ListCollections(userID)
}
func (m *mutexWrapper) CreateCollection(userID model.UserKey, collection model.Collection) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.CreateCollection(userID, collection)
}
func (m *mutexWrapper) RetrieveCollection(userID model.UserKey, collectionID model.CollectionKey) (model.Collection, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveCollection(userID, collectionID)
}
func (m *mutexWrapper) UpdateCollection(userID model.UserKey, collection model.Collection) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateCollection(userID, collection)
}
func (m *mutexWrapper) DeleteCollection(userID model.UserKey, collectionID model.CollectionKey) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.DeleteCollection(userID, collectionID)
}
func (m *mutexWrapper) RetrieveCollectionTags(userID model.UserKey, collectionID string) (model.Tags, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveCollectionTags(userID, collectionID)
}
func (m *mutexWrapper) UpdateCollectionTags(userID model.UserKey, collectionID string, tags model.Tags) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateCollectionTags(userID, collectionID, tags)
}
func (m *mutexWrapper) NewOutputID() model.OutputKey {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.NewOutputID()
}
func (m *mutexWrapper) ListOutputs(userID model.UserKey, collectionID model.CollectionKey) ([]model.Output, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.ListOutputs(userID, collectionID)
}
func (m *mutexWrapper) CreateOutput(userID model.UserKey, output model.Output) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.CreateOutput(userID, output)
}
func (m *mutexWrapper) RetrieveOutput(userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) (model.Output, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveOutput(userID, collectionID, outputID)
}
func (m *mutexWrapper) DeleteOutput(userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.DeleteOutput(userID, collectionID, outputID)
}
func (m *mutexWrapper) UpdateOutput(userID model.UserKey, collectionID model.CollectionKey, output model.Output) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateOutput(userID, collectionID, output)
}
func (m *mutexWrapper) RetrieveOutputTags(userID model.UserKey, outputID string) (model.Tags, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveOutputTags(userID, outputID)
}
func (m *mutexWrapper) UpdateOutputTags(userID model.UserKey, outputID string, tags model.Tags) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateOutputTags(userID, outputID, tags)
}
func (m *mutexWrapper) ListInvites(teamID model.TeamKey, userID model.UserKey) ([]model.Invite, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.ListInvites(teamID, userID)
}
func (m *mutexWrapper) CreateInvite(invite model.Invite) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.CreateInvite(invite)
}
func (m *mutexWrapper) DeleteInvite(code string, teamID model.TeamKey, userID model.UserKey) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.DeleteInvite(code, teamID, userID)
}
func (m *mutexWrapper) AcceptInvite(invite model.Invite, userID model.UserKey) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.AcceptInvite(invite, userID)
}
func (m *mutexWrapper) RetrieveInvite(code string) (model.Invite, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveInvite(code)
}
func (m *mutexWrapper) NewTeamID() model.TeamKey {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.NewTeamID()
}
func (m *mutexWrapper) ListTeams(userID model.UserKey) ([]model.Team, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.ListTeams(userID)
}
func (m *mutexWrapper) CreateTeam(team model.Team) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.CreateTeam(team)
}
func (m *mutexWrapper) RetrieveTeam(userID model.UserKey, teamID model.TeamKey) (model.Team, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveTeam(userID, teamID)
}
func (m *mutexWrapper) DeleteTeam(userID model.UserKey, teamID model.TeamKey) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.DeleteTeam(userID, teamID)
}
func (m *mutexWrapper) UpdateTeam(userID model.UserKey, team model.Team) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateTeam(userID, team)
}
func (m *mutexWrapper) RetrieveTeamTags(userID model.UserKey, teamID string) (model.Tags, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveTeamTags(userID, teamID)
}
func (m *mutexWrapper) UpdateTeamTags(userID model.UserKey, teamID string, tags model.Tags) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateTeamTags(userID, teamID, tags)
}
func (m *mutexWrapper) AllocateSequence(identifier string, current uint64, new uint64) bool {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.AllocateSequence(identifier, current, new)
}

func (m *mutexWrapper) CurrentSequence(identifier string) (uint64, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.CurrentSequence(identifier)
}

func (m *mutexWrapper) OutputListAll() ([]model.Output, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.OutputListAll()
}

func (m *mutexWrapper) NewFirmwareID() model.FirmwareKey {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.NewFirmwareID()
}

func (m *mutexWrapper) CreateFirmware(userID model.UserKey, fw model.Firmware) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.CreateFirmware(userID, fw)
}
func (m *mutexWrapper) RetrieveFirmware(userID model.UserKey, collectionID model.CollectionKey, fwID model.FirmwareKey) (model.Firmware, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveFirmware(userID, collectionID, fwID)
}
func (m *mutexWrapper) DeleteFirmware(userID model.UserKey, collectionID model.CollectionKey, fwID model.FirmwareKey) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.DeleteFirmware(userID, collectionID, fwID)
}
func (m *mutexWrapper) ListFirmware(userID model.UserKey, collectionID model.CollectionKey) ([]model.Firmware, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.ListFirmware(userID, collectionID)
}

func (m *mutexWrapper) UpdateFirmware(userID model.UserKey, collectionID model.CollectionKey, fw model.Firmware) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateFirmware(userID, collectionID, fw)
}

func (m *mutexWrapper) RetrieveFirmwareTags(userID model.UserKey, firmwareID string) (model.Tags, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveFirmwareTags(userID, firmwareID)
}
func (m *mutexWrapper) UpdateFirmwareTags(userID model.UserKey, firmwareID string, tags model.Tags) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateFirmwareTags(userID, firmwareID, tags)
}

func (m *mutexWrapper) RetrieveCurrentAndTargetFirmware(collectionID model.CollectionKey, firmwareA model.FirmwareKey, firmwareB model.FirmwareKey) (model.Firmware, model.Firmware, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveCurrentAndTargetFirmware(collectionID, firmwareA, firmwareB)
}

func (m *mutexWrapper) RetrieveFirmwareConfig(collectionID model.CollectionKey, deviceID model.DeviceKey) (model.FirmwareConfig, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveFirmwareConfig(collectionID, deviceID)
}

func (m *mutexWrapper) RetrieveFirmwareVersionsInUse(userID model.UserKey, collectionID model.CollectionKey, firmwareID model.FirmwareKey) (model.FirmwareUse, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveFirmwareVersionsInUse(userID, collectionID, firmwareID)
}

func (m *mutexWrapper) RetrieveFirmwareByVersion(collectionID model.CollectionKey, version string) (model.Firmware, error) {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.RetrieveFirmwareByVersion(collectionID, version)
}

func (m *mutexWrapper) UpdateFirmwareStateForDevice(imsi int64, state model.DeviceFirmwareState, message string) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.src.UpdateFirmwareStateForDevice(imsi, state, message)
}
