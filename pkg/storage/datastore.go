package storage

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
	"github.com/eesrc/horde/pkg/model"
)

// DataStore is the data store used by the REST API
type DataStore interface {
	// RetrieveUserByExternalID retrieves the user with the specific external ID.
	RetrieveUserByExternalID(string, model.AuthMethod) (*model.User, error)
	// NewUserID returns a new unique user ID each time it is called.
	NewUserID() model.UserKey
	// CreateUser crates a new user in the backend store.
	CreateUser(user model.User, privateTeam model.Team) error
	// UpdateUser updates the user in the backend store.
	UpdateUser(*model.User) error
	// RetreiveUser retrieves a single user from its ID
	RetrieveUser(model.UserKey) (model.User, error)

	// CreateToken creates a new access token in the backend store.
	CreateToken(model.Token) error
	// ListTokens lists all tokens available for the specified user. If the user
	// is an administrator of the team owning the tokens both read-only and
	// read/write tokens will be returned. Regular members will only see
	// read-only tokens.
	ListTokens(userID model.UserKey) ([]model.Token, error)
	// UpdateToken updates tokens in the backend store. The user must be the
	// owner of the token.
	UpdateToken(model.Token) error
	// DeleteToken removes a token from the backend store. The user must be the
	// owner of the token.
	DeleteToken(userID model.UserKey, token string) error
	// RetrieveTokenTags retrieves the tags for the token. The user must be
	// the owner of the token.
	RetrieveTokenTags(userID model.UserKey, token string) (model.Tags, error)
	// UpdateTokenTags updates the tags on the token. The user must be owner
	// of the token.
	UpdateTokenTags(userID model.UserKey, token string, tags model.Tags) error

	// Retrieve token retrieves a single token
	RetrieveToken(token string) (model.Token, error)

	// DeviceNewID creates a new device ID. The returned keys are unique for
	// every call to DeviceNewID.
	NewDeviceID() model.DeviceKey

	// CreateDevice creates a new device. The user must be an team admin for
	// the team that owns the device.
	CreateDevice(userID model.UserKey, newDevice model.Device) error

	// ListDevices lists all devices in a particular collection. The user must
	// be a member of the team owning the collection.
	ListDevices(userID model.UserKey, collectionID model.CollectionKey) ([]model.Device, error)

	// RetrieveDevice retrieves a single device. The user must be a member of
	// the team that owns the device.
	RetrieveDevice(userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) (model.Device, error)

	// DeleteDevice removes a device from the backend store. The user must be
	// an admin of the team that owns the collection the device is stored in.
	DeleteDevice(userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) error

	// UpdateDevice updates a device in the backend store. The user must be an
	// admin of the team that owns the device's collection.
	UpdateDevice(userID model.UserKey, collectionID model.CollectionKey, device model.Device) error

	// RetrieveDeviceTags retrieves tags for a particular device. The user must
	// be a member of the team that owns the device.
	RetrieveDeviceTags(userID model.UserKey, deviceID string) (model.Tags, error)

	// UpdateDeviceTags updates the tags on a particular device. The user must
	// be an admin of the team that owns the device.
	UpdateDeviceTags(userID model.UserKey, deviceID string, tags model.Tags) error

	// RetrieveDeviceByIMSI retrieves a device based on the device's
	// IMSI. The device must be allocated an IP address to be returned.
	RetrieveDeviceByIMSI(imsi int64) (model.Device, error)

	// RetrieveDeviceByMSISDN retrieves a device based on its MSISDN.
	// This field isn't a part of the device and uses an external lookup table
	// (or system) to determine which device this is.
	RetrieveDeviceByMSISDN(msisdn string) (model.Device, error)

	// UpdateDeviceMetadata updates tags on a device. The method will update any
	// device in the store.
	UpdateDeviceMetadata(device model.Device) error

	// NewCollectionID creates a new ID for a collection. It should be unique
	// for the keygenerator used
	NewCollectionID() model.CollectionKey
	// ListCollections lists all collections available to the the given user. The
	// user must be a member of the team owning the collection for it to show up
	// in the list.
	ListCollections(userID model.UserKey) ([]model.Collection, error)
	// CreateCollection creates a new collection. If the collection already exists
	// it will return storage.ErrAlreadyExists.
	CreateCollection(userID model.UserKey, collection model.Collection) error
	// RetrieveCollection retrieves a collection in the scope of the supplied user.
	// if the user can't access the collection or it doesn't exist it returns
	// storage.ErrNotFound
	RetrieveCollection(userID model.UserKey, collectionID model.CollectionKey) (model.Collection, error)
	// UpdateCollection updates the collection. The user must be an admin of the
	// team owning the colleciton. If the user isn't an admin storage.ErrAccess
	// is returned. storage.ErrNotFound is returned if the user isn't a member of
	// the owning team.
	UpdateCollection(userID model.UserKey, collection model.Collection) error
	// DeleteCollection removes a collection. The collection must be empty of
	// devices and outputs, otherwise storage.ErrConflict is returned.
	// The user must be an administrator of the team that owns the collection,
	// otherwise storage.ErrAccess (if he/she is a member) or storage.ErrNotFound (if
	// he/she isn't) is returned.
	DeleteCollection(userID model.UserKey, collectionID model.CollectionKey) error
	// RetrieveCollectionTags retrieves tags for a collection. The user must be
	// a member of the team owning the collection.
	RetrieveCollectionTags(userID model.UserKey, collectionID string) (model.Tags, error)
	// UpdateCollectionTags updates tags for a collection. The user must be an
	// administrator in the team owning the collection.
	UpdateCollectionTags(userID model.UserKey, collectionID string, tags model.Tags) error

	// NewOutputID creates a new ID for outputs. Every time OutputNewID is
	// called it will return a new unique ID.
	NewOutputID() model.OutputKey

	// ListOutputs returns a list of outputs for a particular collection. The
	// user must be a member of the team owning the collecion. If the
	// collection can't be found (or if the user isn't a member of the team
	// owning the collection) it will return storage.ErrNotFound.
	ListOutputs(userID model.UserKey, collectionID model.CollectionKey) ([]model.Output, error)

	// CreateOutput creates a new output for the specified collection. The
	// user must be an administrator in the team owning the collection. If the
	// user isn't an admin or the collection doesn't exist it will return
	// storage.ErrNotFound.
	CreateOutput(userID model.UserKey, output model.Output) error

	// RetrieveOutput returns a single output. The user must be a member of the
	// team owning the collection (and implicitly the output).
	RetrieveOutput(userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) (model.Output, error)

	// DeleteOutput removes the output. The user must be an admin of the team
	// that owns the collection.
	DeleteOutput(userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) error

	// UpdateOutput updates the output. The user must be an admin of the team
	// that owns the collection.
	UpdateOutput(userID model.UserKey, collectionID model.CollectionKey, output model.Output) error

	// RetriveOutputTags returns the tags for the spcified output.
	RetrieveOutputTags(userID model.UserKey, outputID string) (model.Tags, error)

	// UpdateOutputTags updates the tags for the output.
	UpdateOutputTags(userID model.UserKey, outputID string, tags model.Tags) error

	// OutputListAll lists all outputs in the data store
	OutputListAll() ([]model.Output, error)

	// ListInvites returns a list of invites for a particular team. An unknown
	// team will return NotFound. The user must be an administrator of the
	// team to view invites.
	ListInvites(teamID model.TeamKey, userID model.UserKey) ([]model.Invite, error)
	// CreateInvite creates a new invite. The user must be an administrator of
	// the team. If the invite cannot be created it will return an error.
	// Invite codes must be unique.
	CreateInvite(invite model.Invite) error
	// DeleteInvite removes an invite. The user must be an admin of the team.
	DeleteInvite(code string, teamID model.TeamKey, userID model.UserKey) error
	// AcceptInvite accepts an invite. Returns an error if the invite doesn't
	// exist or if the user is already a member of the team.
	AcceptInvite(invite model.Invite, userID model.UserKey) error
	// RetrieveInvite retrieves an invite. Returns an error if there's no invite
	// with that particular code. Does not check if the code is valid.
	RetrieveInvite(code string) (model.Invite, error)

	// NewTeamID creates a new team ID in the backend store. Each call returns
	// a new unique ID.
	NewTeamID() model.TeamKey
	// ListTeams returns a list of teams that the user is a member of.
	ListTeams(userID model.UserKey) ([]model.Team, error)
	// CreateTeam creates a new team. The member list must be populated with at
	// least one administrator.
	CreateTeam(team model.Team) error
	// RetrieveTeam returns the team in the backend store with the given
	// team ID. The user must be a member of the team. Returns storage.ErrNotFound
	// if the team doesn't exist or if the user isn't a member of the team.
	RetrieveTeam(userID model.UserKey, teamID model.TeamKey) (model.Team, error)
	// DeleteTeam removes the team with the given team ID. The user must be
	// an administrator of the team.
	DeleteTeam(userID model.UserKey, teamID model.TeamKey) error
	// UpdateTeam updates the team. The user must be an administrator of the
	// team. Returns storage.ErrNoAccess if the user is just a plain member,
	// storage.ErrNotFound if the team doesn't exist or the user isn't a member
	// of the team.
	UpdateTeam(userID model.UserKey, team model.Team) error
	// RetrieveTeamTags returns the tags for the team. The user must be a
	// member of the team.
	RetrieveTeamTags(userID model.UserKey, teamID string) (model.Tags, error)
	// UpdateTeamTags updates the tags on the team. The user must be an
	// administrator of the team.
	UpdateTeamTags(userID model.UserKey, teamID string, tags model.Tags) error

	// NewFirmwareID creates a new identifier for firmware metadata
	NewFirmwareID() model.FirmwareKey

	// CreateFirmware creates a new firmware image reference
	CreateFirmware(userID model.UserKey, fwID model.Firmware) error
	// RetrieveFirmware retrieves firmware metadata
	RetrieveFirmware(userID model.UserKey, collectionID model.CollectionKey, fwID model.FirmwareKey) (model.Firmware, error)
	// DeleteFirmware removes a firmware reference. It does not remove the firmware image itself
	DeleteFirmware(userID model.UserKey, collectionID model.CollectionKey, fwID model.FirmwareKey) error
	// ListFirmware lists all firmware images owned by teams the user is a member of
	ListFirmware(userID model.UserKey, collectionID model.CollectionKey) ([]model.Firmware, error)
	// UpdateFirmware updates the firmware image. The only fields that can be updated are
	// TeamID and tags.
	UpdateFirmware(userID model.UserKey, collectionID model.CollectionKey, fw model.Firmware) error

	// RetrieveFirmwareTags returns the tags for the firmware image. The user must be a
	// member of the team owning the firmware
	RetrieveFirmwareTags(userID model.UserKey, firmwareID string) (model.Tags, error)
	// UpdateFirmwareTags updates the tags on the firmware image. The user must be an
	// administrator of the team owning the firmware
	UpdateFirmwareTags(userID model.UserKey, firmwareID string, tags model.Tags) error

	// RetriveFirmwareConfig retrieves the firmware config for a particular device. The
	// devices firmware config might be overridden by the collection so both have
	// to be checked.
	RetrieveFirmwareConfig(collectionID model.CollectionKey, deviceID model.DeviceKey) (model.FirmwareConfig, error)

	// RetrieveCurrentAndTargetFirmware retrieves two versions. There is no check on collection here
	RetrieveCurrentAndTargetFirmware(collectionID model.CollectionKey, firmwareA model.FirmwareKey, firmwareB model.FirmwareKey) (model.Firmware, model.Firmware, error)

	// RetrieveFirmwareVersionsInUse returns the devices that either have the firmware currently in use or are targeted with the firmware version.
	RetrieveFirmwareVersionsInUse(userID model.UserKey, collectionID model.CollectionKey, firmwareID model.FirmwareKey) (model.FirmwareUse, error)

	// RetrieveFirmwareByVersion returns the firmware ID for the version. The
	// version is unique for the collection. If the firmware isn't found storage.ErrNotFound is returned.
	RetrieveFirmwareByVersion(collectionID model.CollectionKey, version string) (model.Firmware, error)

	// UpdateFirmwareStateForDevice updates the firmware state field for a device
	UpdateFirmwareStateForDevice(imsi int64, state model.DeviceFirmwareState, message string) error

	SequenceStore
}
