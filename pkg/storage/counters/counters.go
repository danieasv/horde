// Package counters contains a storage wrapper that updates the
// performance counters for the horde core service.
package counters

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
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

type counterWrapStore struct {
	store storage.DataStore
}

// NewCounterWrapperStore creates a wrapper for an existing storage.DataStore
// that updates performance counters wrt new users, devices, outputs, teams
// and so on
func NewCounterWrapperStore(ds storage.DataStore) storage.DataStore {
	return &counterWrapStore{store: ds}
}

func (c *counterWrapStore) CreateUser(user model.User, privateTeam model.Team) error {
	err := c.store.CreateUser(user, privateTeam)
	if err == nil {
		metrics.DefaultCoreCounters.UserCount.Inc()
	}
	return err
}
func (c *counterWrapStore) CreateDevice(userID model.UserKey, newDevice model.Device) error {
	err := c.store.CreateDevice(userID, newDevice)
	if err == nil {
		metrics.DefaultCoreCounters.DeviceCount.Inc()
	}
	return err
}
func (c *counterWrapStore) DeleteDevice(userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) error {
	err := c.store.DeleteDevice(userID, collectionID, deviceID)
	if err == nil {
		metrics.DefaultCoreCounters.DeviceCount.Dec()
	}
	return err
}
func (c *counterWrapStore) CreateCollection(userID model.UserKey, collection model.Collection) error {
	err := c.store.CreateCollection(userID, collection)
	if err == nil {
		metrics.DefaultCoreCounters.CollectionCount.Inc()
	}
	return err
}
func (c *counterWrapStore) DeleteCollection(userID model.UserKey, collectionID model.CollectionKey) error {
	err := c.store.DeleteCollection(userID, collectionID)
	if err == nil {
		metrics.DefaultCoreCounters.CollectionCount.Dec()
	}
	return err
}
func (c *counterWrapStore) CreateOutput(userID model.UserKey, output model.Output) error {
	err := c.store.CreateOutput(userID, output)
	if err == nil {
		metrics.DefaultCoreCounters.OutputCount.Inc()
	}
	return err
}
func (c *counterWrapStore) DeleteOutput(userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) error {
	err := c.store.DeleteOutput(userID, collectionID, outputID)
	if err == nil {
		metrics.DefaultCoreCounters.OutputCount.Dec()
	}
	return err
}
func (c *counterWrapStore) CreateTeam(team model.Team) error {
	err := c.store.CreateTeam(team)
	if err == nil {
		metrics.DefaultCoreCounters.TeamCount.Inc()
	}
	return err
}
func (c *counterWrapStore) DeleteTeam(userID model.UserKey, teamID model.TeamKey) error {
	err := c.store.DeleteTeam(userID, teamID)
	if err == nil {
		metrics.DefaultCoreCounters.TeamCount.Dec()
	}
	return err
}
func (c *counterWrapStore) CreateInvite(invite model.Invite) error {
	err := c.store.CreateInvite(invite)
	if err == nil {
		metrics.DefaultCoreCounters.InvitesCreated.Inc()
	}
	return err
}
func (c *counterWrapStore) AcceptInvite(invite model.Invite, userID model.UserKey) error {
	err := c.store.AcceptInvite(invite, userID)
	if err == nil {
		metrics.DefaultCoreCounters.InvitesAccepted.Inc()
	}
	return err
}

//
// The rest of the methods just calls the wrapped store directly
// ---------------------------------------------------------------------------
func (c *counterWrapStore) RetrieveUserByExternalID(id string, auth model.AuthMethod) (*model.User, error) {
	return c.store.RetrieveUserByExternalID(id, auth)
}

func (c *counterWrapStore) NewUserID() model.UserKey {
	return c.store.NewUserID()
}

func (c *counterWrapStore) UpdateUser(user *model.User) error {
	return c.store.UpdateUser(user)
}

func (c *counterWrapStore) RetrieveUser(id model.UserKey) (model.User, error) {
	return c.store.RetrieveUser(id)
}

func (c *counterWrapStore) CreateToken(token model.Token) error {
	return c.store.CreateToken(token)
}

func (c *counterWrapStore) ListTokens(userID model.UserKey) ([]model.Token, error) {
	return c.store.ListTokens(userID)
}

func (c *counterWrapStore) UpdateToken(token model.Token) error {
	return c.store.UpdateToken(token)
}

func (c *counterWrapStore) DeleteToken(userID model.UserKey, token string) error {
	return c.store.DeleteToken(userID, token)
}

func (c *counterWrapStore) RetrieveTokenTags(userID model.UserKey, token string) (model.Tags, error) {
	return c.store.RetrieveTokenTags(userID, token)
}

func (c *counterWrapStore) UpdateTokenTags(userID model.UserKey, token string, tags model.Tags) error {
	return c.store.UpdateTokenTags(userID, token, tags)
}

func (c *counterWrapStore) RetrieveToken(token string) (model.Token, error) {
	return c.store.RetrieveToken(token)
}

func (c *counterWrapStore) NewDeviceID() model.DeviceKey {
	return c.store.NewDeviceID()
}

func (c *counterWrapStore) ListDevices(userID model.UserKey, collectionID model.CollectionKey) ([]model.Device, error) {
	return c.store.ListDevices(userID, collectionID)
}

func (c *counterWrapStore) RetrieveDevice(userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) (model.Device, error) {
	return c.store.RetrieveDevice(userID, collectionID, deviceID)
}

func (c *counterWrapStore) UpdateDevice(userID model.UserKey, collectionID model.CollectionKey, device model.Device) error {
	return c.store.UpdateDevice(userID, collectionID, device)
}

func (c *counterWrapStore) RetrieveDeviceTags(userID model.UserKey, deviceID string) (model.Tags, error) {
	return c.store.RetrieveDeviceTags(userID, deviceID)
}

func (c *counterWrapStore) UpdateDeviceTags(userID model.UserKey, deviceID string, tags model.Tags) error {
	return c.store.UpdateDeviceTags(userID, deviceID, tags)
}

func (c *counterWrapStore) RetrieveDeviceByIMSI(imsi int64) (model.Device, error) {
	return c.store.RetrieveDeviceByIMSI(imsi)
}

func (c *counterWrapStore) RetrieveDeviceByMSISDN(msisdn string) (model.Device, error) {
	return c.store.RetrieveDeviceByMSISDN(msisdn)
}

func (c *counterWrapStore) UpdateDeviceMetadata(device model.Device) error {
	return c.store.UpdateDeviceMetadata(device)
}

func (c *counterWrapStore) NewCollectionID() model.CollectionKey {
	return c.store.NewCollectionID()
}

func (c *counterWrapStore) ListCollections(userID model.UserKey) ([]model.Collection, error) {
	return c.store.ListCollections(userID)
}

func (c *counterWrapStore) RetrieveCollection(userID model.UserKey, collectionID model.CollectionKey) (model.Collection, error) {
	return c.store.RetrieveCollection(userID, collectionID)
}

func (c *counterWrapStore) UpdateCollection(userID model.UserKey, collection model.Collection) error {
	return c.store.UpdateCollection(userID, collection)
}

func (c *counterWrapStore) RetrieveCollectionTags(userID model.UserKey, collectionID string) (model.Tags, error) {
	return c.store.RetrieveCollectionTags(userID, collectionID)
}

func (c *counterWrapStore) UpdateCollectionTags(userID model.UserKey, collectionID string, tags model.Tags) error {
	return c.store.UpdateCollectionTags(userID, collectionID, tags)
}

func (c *counterWrapStore) NewOutputID() model.OutputKey {
	return c.store.NewOutputID()
}

func (c *counterWrapStore) ListOutputs(userID model.UserKey, collectionID model.CollectionKey) ([]model.Output, error) {
	return c.store.ListOutputs(userID, collectionID)
}

func (c *counterWrapStore) RetrieveOutput(userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) (model.Output, error) {
	return c.store.RetrieveOutput(userID, collectionID, outputID)
}

func (c *counterWrapStore) UpdateOutput(userID model.UserKey, collectionID model.CollectionKey, output model.Output) error {
	return c.store.UpdateOutput(userID, collectionID, output)
}

func (c *counterWrapStore) RetrieveOutputTags(userID model.UserKey, outputID string) (model.Tags, error) {
	return c.store.RetrieveOutputTags(userID, outputID)
}

func (c *counterWrapStore) UpdateOutputTags(userID model.UserKey, outputID string, tags model.Tags) error {
	return c.store.UpdateOutputTags(userID, outputID, tags)
}

func (c *counterWrapStore) OutputListAll() ([]model.Output, error) {
	return c.store.OutputListAll()
}

func (c *counterWrapStore) ListInvites(teamID model.TeamKey, userID model.UserKey) ([]model.Invite, error) {
	return c.store.ListInvites(teamID, userID)
}

func (c *counterWrapStore) DeleteInvite(code string, teamID model.TeamKey, userID model.UserKey) error {
	return c.store.DeleteInvite(code, teamID, userID)
}

func (c *counterWrapStore) RetrieveInvite(code string) (model.Invite, error) {
	return c.store.RetrieveInvite(code)
}

func (c *counterWrapStore) NewTeamID() model.TeamKey {
	return c.store.NewTeamID()
}

func (c *counterWrapStore) ListTeams(userID model.UserKey) ([]model.Team, error) {
	return c.store.ListTeams(userID)
}

func (c *counterWrapStore) RetrieveTeam(userID model.UserKey, teamID model.TeamKey) (model.Team, error) {
	return c.store.RetrieveTeam(userID, teamID)
}

func (c *counterWrapStore) UpdateTeam(userID model.UserKey, team model.Team) error {
	return c.store.UpdateTeam(userID, team)
}

func (c *counterWrapStore) RetrieveTeamTags(userID model.UserKey, teamID string) (model.Tags, error) {
	return c.store.RetrieveTeamTags(userID, teamID)
}

func (c *counterWrapStore) UpdateTeamTags(userID model.UserKey, teamID string, tags model.Tags) error {
	return c.store.UpdateTeamTags(userID, teamID, tags)
}

func (c *counterWrapStore) AllocateSequence(identifier string, current uint64, new uint64) bool {
	return c.store.AllocateSequence(identifier, current, new)
}
func (c *counterWrapStore) CurrentSequence(identifier string) (uint64, error) {
	return c.store.CurrentSequence(identifier)
}
func (c *counterWrapStore) NewFirmwareID() model.FirmwareKey {
	return c.store.NewFirmwareID()
}
func (c *counterWrapStore) CreateFirmware(userID model.UserKey, fw model.Firmware) error {
	return c.store.CreateFirmware(userID, fw)
}
func (c *counterWrapStore) RetrieveFirmware(userID model.UserKey, collectionID model.CollectionKey, fwID model.FirmwareKey) (model.Firmware, error) {
	return c.store.RetrieveFirmware(userID, collectionID, fwID)
}
func (c *counterWrapStore) DeleteFirmware(userID model.UserKey, collectionID model.CollectionKey, fwID model.FirmwareKey) error {
	return c.store.DeleteFirmware(userID, collectionID, fwID)
}
func (c *counterWrapStore) ListFirmware(userID model.UserKey, collectionID model.CollectionKey) ([]model.Firmware, error) {
	return c.store.ListFirmware(userID, collectionID)
}
func (c *counterWrapStore) UpdateFirmware(userID model.UserKey, collectionID model.CollectionKey, fw model.Firmware) error {
	return c.store.UpdateFirmware(userID, collectionID, fw)
}
func (c *counterWrapStore) RetrieveFirmwareTags(userID model.UserKey, firmwareID string) (model.Tags, error) {
	return c.store.RetrieveFirmwareTags(userID, firmwareID)
}
func (c *counterWrapStore) UpdateFirmwareTags(userID model.UserKey, firmwareID string, tags model.Tags) error {
	return c.store.UpdateFirmwareTags(userID, firmwareID, tags)
}
func (c *counterWrapStore) RetrieveCurrentAndTargetFirmware(collectionID model.CollectionKey, firmwareA model.FirmwareKey, firmwareB model.FirmwareKey) (model.Firmware, model.Firmware, error) {
	return c.store.RetrieveCurrentAndTargetFirmware(collectionID, firmwareA, firmwareB)
}
func (c *counterWrapStore) RetrieveFirmwareConfig(collectionID model.CollectionKey, deviceID model.DeviceKey) (model.FirmwareConfig, error) {
	return c.store.RetrieveFirmwareConfig(collectionID, deviceID)
}
func (c *counterWrapStore) RetrieveFirmwareVersionsInUse(userID model.UserKey, collectionID model.CollectionKey, firmwareID model.FirmwareKey) (model.FirmwareUse, error) {
	return c.store.RetrieveFirmwareVersionsInUse(userID, collectionID, firmwareID)
}
func (c *counterWrapStore) RetrieveFirmwareByVersion(collectionID model.CollectionKey, version string) (model.Firmware, error) {
	return c.store.RetrieveFirmwareByVersion(collectionID, version)
}
func (c *counterWrapStore) UpdateFirmwareStateForDevice(imsi int64, state model.DeviceFirmwareState, message string) error {
	return c.store.UpdateFirmwareStateForDevice(imsi, state, message)
}
