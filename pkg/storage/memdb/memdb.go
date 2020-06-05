package memdb

import (
	"net"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// newMemCacheDB creates a cached database
func newCacheDB(persistent, memory storage.DataStore) storage.DataStore {
	return &memoryDB{
		persistent:    persistent,
		inmem:         memory,
		persistentAPN: nil,
		inmemAPN:      nil,
	}
}

func newAPNCacheDB(persistent, memory storage.APNStore) storage.APNStore {
	return &memoryDB{
		persistent:    nil,
		inmem:         nil,
		persistentAPN: persistent,
		inmemAPN:      memory,
	}
}

// This is a very simple implementation that basically uses two storage
// implementations, one in memory and one persistent. The persistent store will
// typically be orders of magnitude slower than the in memory implementation
// and this (meta-) type will just update the persistent store first, then the
// in memory store. Retrieval will only use the in memory store. The persistent
// store is always updated first -- if that fails there's no point in checking
// the in memory store. If the two stores becomes out of sync (ie a
// configuration error where the persistent store is updated from elsewhere
// we'll see weird errors...but the persistent store will be consistent.
//
// Right now we err on the side of caution and panic whenever something is out
// of sync. It's horrible if you have a single instance, a little less horrible
// if you are running a cluster.
//
// The in memory database must be primed by reading the (relevant) data from the
// persistent storage
type memoryDB struct {
	persistent    storage.DataStore
	inmem         storage.DataStore
	persistentAPN storage.APNStore
	inmemAPN      storage.APNStore
}

func (m *memoryDB) CreateToken(token model.Token) error {
	// Create in persisted storage first
	if err := m.persistent.CreateToken(token); err != nil {
		return err
	}
	if err := m.inmem.CreateToken(token); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) ListTokens(userID model.UserKey) ([]model.Token, error) {
	return m.inmem.ListTokens(userID)
}

func (m *memoryDB) UpdateToken(token model.Token) error {
	if err := m.persistent.UpdateToken(token); err != nil {
		return err
	}
	if err := m.inmem.UpdateToken(token); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) DeleteToken(userID model.UserKey, token string) error {
	if err := m.persistent.DeleteToken(userID, token); err != nil {
		return err
	}
	if err := m.inmem.DeleteToken(userID, token); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveTokenTags(userID model.UserKey, token string) (model.Tags, error) {
	return m.inmem.RetrieveTokenTags(userID, token)
}

func (m *memoryDB) UpdateTokenTags(userID model.UserKey, token string, tags model.Tags) error {
	if err := m.persistent.UpdateTokenTags(userID, token, tags); err != nil {
		return err
	}
	if err := m.inmem.UpdateTokenTags(userID, token, tags); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveToken(token string) (model.Token, error) {
	return m.inmem.RetrieveToken(token)
}

func (m *memoryDB) ListInvites(teamID model.TeamKey, userID model.UserKey) ([]model.Invite, error) {
	return m.inmem.ListInvites(teamID, userID)
}

func (m *memoryDB) CreateInvite(invite model.Invite) error {
	if err := m.persistent.CreateInvite(invite); err != nil {
		return err
	}
	if err := m.inmem.CreateInvite(invite); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) DeleteInvite(code string, teamID model.TeamKey, userID model.UserKey) error {
	if err := m.persistent.DeleteInvite(code, teamID, userID); err != nil {
		return err
	}
	if err := m.inmem.DeleteInvite(code, teamID, userID); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) AcceptInvite(invite model.Invite, userID model.UserKey) error {
	if err := m.persistent.AcceptInvite(invite, userID); err != nil {
		return err
	}
	if err := m.inmem.AcceptInvite(invite, userID); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveInvite(code string) (model.Invite, error) {
	return m.inmem.RetrieveInvite(code)
}

// Sequences will always mutate in the persistent store -- for obvious reasons
func (m *memoryDB) AllocateSequence(identifier string, current uint64, new uint64) bool {
	return m.persistent.AllocateSequence(identifier, current, new)
}

// Retrieving the current sequence is mostly a one off - for obvious reasons
func (m *memoryDB) CurrentSequence(identifier string) (uint64, error) {
	return m.persistent.CurrentSequence(identifier)
}

// Also huge note to self: Make sure key generator works for shards. Need to use the
// asigned shard and block new keys when not in a quorum.

func (m *memoryDB) NewCollectionID() model.CollectionKey {
	// just return the persistent implementation here.
	return m.persistent.NewCollectionID()
}

func (m *memoryDB) ListCollections(userID model.UserKey) ([]model.Collection, error) {
	return m.inmem.ListCollections(userID)
}

func (m *memoryDB) CreateCollection(userID model.UserKey, collection model.Collection) error {
	if err := m.persistent.CreateCollection(userID, collection); err != nil {
		return err
	}
	if err := m.inmem.CreateCollection(userID, collection); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveCollection(userID model.UserKey, collectionID model.CollectionKey) (model.Collection, error) {
	return m.inmem.RetrieveCollection(userID, collectionID)
}

func (m *memoryDB) UpdateCollection(userID model.UserKey, collection model.Collection) error {
	if err := m.persistent.UpdateCollection(userID, collection); err != nil {
		return err
	}
	if err := m.inmem.UpdateCollection(userID, collection); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) DeleteCollection(userID model.UserKey, collectionID model.CollectionKey) error {
	if err := m.persistent.DeleteCollection(userID, collectionID); err != nil {
		return err
	}
	if err := m.inmem.DeleteCollection(userID, collectionID); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveCollectionTags(userID model.UserKey, collectionID string) (model.Tags, error) {
	return m.inmem.RetrieveCollectionTags(userID, collectionID)
}

func (m *memoryDB) UpdateCollectionTags(userID model.UserKey, collectionID string, tags model.Tags) error {
	if err := m.persistent.UpdateCollectionTags(userID, collectionID, tags); err != nil {
		return err
	}
	if err := m.inmem.UpdateCollectionTags(userID, collectionID, tags); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) NewDeviceID() model.DeviceKey {
	return m.persistent.NewDeviceID()
}

func (m *memoryDB) CreateDevice(userID model.UserKey, newDevice model.Device) error {
	if err := m.persistent.CreateDevice(userID, newDevice); err != nil {
		return err
	}
	if err := m.inmem.CreateDevice(userID, newDevice); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) ListDevices(userID model.UserKey, collectionID model.CollectionKey) ([]model.Device, error) {
	return m.inmem.ListDevices(userID, collectionID)
}

func (m *memoryDB) RetrieveDevice(userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) (model.Device, error) {
	return m.inmem.RetrieveDevice(userID, collectionID, deviceID)
}

func (m *memoryDB) DeleteDevice(userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) error {
	if err := m.persistent.DeleteDevice(userID, collectionID, deviceID); err != nil {
		return err
	}
	if err := m.inmem.DeleteDevice(userID, collectionID, deviceID); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) UpdateDevice(userID model.UserKey, collectionID model.CollectionKey, device model.Device) error {
	if err := m.persistent.UpdateDevice(userID, collectionID, device); err != nil {
		return err
	}
	if err := m.inmem.UpdateDevice(userID, collectionID, device); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveDeviceTags(userID model.UserKey, deviceID string) (model.Tags, error) {
	return m.inmem.RetrieveDeviceTags(userID, deviceID)
}

func (m *memoryDB) UpdateDeviceTags(userID model.UserKey, deviceID string, tags model.Tags) error {
	if err := m.persistent.UpdateDeviceTags(userID, deviceID, tags); err != nil {
		return err
	}
	if err := m.inmem.UpdateDeviceTags(userID, deviceID, tags); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveDeviceByIMSI(imsi int64) (model.Device, error) {
	return m.inmem.RetrieveDeviceByIMSI(imsi)
}

func (m *memoryDB) RetrieveDeviceByMSISDN(msisdn string) (model.Device, error) {
	return m.inmem.RetrieveDeviceByMSISDN(msisdn)
}

func (m *memoryDB) UpdateDeviceMetadata(device model.Device) error {
	if err := m.persistent.UpdateDeviceMetadata(device); err != nil {
		return err
	}
	if err := m.inmem.UpdateDeviceMetadata(device); err != nil {
		return err
	}
	return nil
}

func (m *memoryDB) NewFirmwareID() model.FirmwareKey {
	return m.persistent.NewFirmwareID()
}

func (m *memoryDB) CreateFirmware(userID model.UserKey, fwID model.Firmware) error {
	if err := m.persistent.CreateFirmware(userID, fwID); err != nil {
		return err
	}
	if err := m.inmem.CreateFirmware(userID, fwID); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveFirmware(userID model.UserKey, collectionID model.CollectionKey, fwID model.FirmwareKey) (model.Firmware, error) {
	return m.inmem.RetrieveFirmware(userID, collectionID, fwID)
}

func (m *memoryDB) DeleteFirmware(userID model.UserKey, collectionID model.CollectionKey, fwID model.FirmwareKey) error {
	if err := m.persistent.DeleteFirmware(userID, collectionID, fwID); err != nil {
		return err
	}
	if err := m.inmem.DeleteFirmware(userID, collectionID, fwID); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) ListFirmware(userID model.UserKey, collectionID model.CollectionKey) ([]model.Firmware, error) {
	return m.inmem.ListFirmware(userID, collectionID)
}

func (m *memoryDB) UpdateFirmware(userID model.UserKey, collectionID model.CollectionKey, fw model.Firmware) error {
	if err := m.persistent.UpdateFirmware(userID, collectionID, fw); err != nil {
		return err
	}
	if err := m.inmem.UpdateFirmware(userID, collectionID, fw); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveFirmwareTags(userID model.UserKey, firmwareID string) (model.Tags, error) {
	return m.inmem.RetrieveFirmwareTags(userID, firmwareID)
}

func (m *memoryDB) UpdateFirmwareTags(userID model.UserKey, firmwareID string, tags model.Tags) error {
	if err := m.persistent.UpdateFirmwareTags(userID, firmwareID, tags); err != nil {
		return err
	}
	if err := m.inmem.UpdateFirmwareTags(userID, firmwareID, tags); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveFirmwareConfig(collectionID model.CollectionKey, deviceID model.DeviceKey) (model.FirmwareConfig, error) {
	return m.inmem.RetrieveFirmwareConfig(collectionID, deviceID)
}

func (m *memoryDB) RetrieveCurrentAndTargetFirmware(collectionID model.CollectionKey, firmwareA model.FirmwareKey, firmwareB model.FirmwareKey) (model.Firmware, model.Firmware, error) {
	return m.inmem.RetrieveCurrentAndTargetFirmware(collectionID, firmwareA, firmwareB)
}

func (m *memoryDB) RetrieveFirmwareVersionsInUse(userID model.UserKey, collectionID model.CollectionKey, firmwareID model.FirmwareKey) (model.FirmwareUse, error) {
	return m.inmem.RetrieveFirmwareVersionsInUse(userID, collectionID, firmwareID)
}

func (m *memoryDB) RetrieveFirmwareByVersion(collectionID model.CollectionKey, version string) (model.Firmware, error) {
	return m.inmem.RetrieveFirmwareByVersion(collectionID, version)
}

func (m *memoryDB) UpdateFirmwareStateForDevice(imsi int64, state model.DeviceFirmwareState, message string) error {
	if err := m.persistent.UpdateFirmwareStateForDevice(imsi, state, message); err != nil {
		return err
	}
	if err := m.inmem.UpdateFirmwareStateForDevice(imsi, state, message); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) NewOutputID() model.OutputKey {
	return m.persistent.NewOutputID()
}

func (m *memoryDB) ListOutputs(userID model.UserKey, collectionID model.CollectionKey) ([]model.Output, error) {
	return m.inmem.ListOutputs(userID, collectionID)
}

func (m *memoryDB) CreateOutput(userID model.UserKey, output model.Output) error {
	if err := m.persistent.CreateOutput(userID, output); err != nil {
		return err
	}
	if err := m.inmem.CreateOutput(userID, output); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveOutput(userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) (model.Output, error) {
	return m.inmem.RetrieveOutput(userID, collectionID, outputID)
}

func (m *memoryDB) DeleteOutput(userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) error {
	if err := m.persistent.DeleteOutput(userID, collectionID, outputID); err != nil {
		return err
	}
	if err := m.inmem.DeleteOutput(userID, collectionID, outputID); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) UpdateOutput(userID model.UserKey, collectionID model.CollectionKey, output model.Output) error {
	if err := m.persistent.UpdateOutput(userID, collectionID, output); err != nil {
		return err
	}
	if err := m.inmem.UpdateOutput(userID, collectionID, output); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveOutputTags(userID model.UserKey, outputID string) (model.Tags, error) {
	return m.inmem.RetrieveOutputTags(userID, outputID)
}

func (m *memoryDB) UpdateOutputTags(userID model.UserKey, outputID string, tags model.Tags) error {
	if err := m.persistent.UpdateOutputTags(userID, outputID, tags); err != nil {
		return err
	}
	if err := m.inmem.UpdateOutputTags(userID, outputID, tags); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) OutputListAll() ([]model.Output, error) {
	return m.inmem.OutputListAll()
}

func (m *memoryDB) NewTeamID() model.TeamKey {
	return m.persistent.NewTeamID()
}

func (m *memoryDB) ListTeams(userID model.UserKey) ([]model.Team, error) {
	return m.inmem.ListTeams(userID)
}

func (m *memoryDB) CreateTeam(team model.Team) error {
	if err := m.persistent.CreateTeam(team); err != nil {
		return err
	}
	if err := m.inmem.CreateTeam(team); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveTeam(userID model.UserKey, teamID model.TeamKey) (model.Team, error) {
	return m.inmem.RetrieveTeam(userID, teamID)
}

func (m *memoryDB) DeleteTeam(userID model.UserKey, teamID model.TeamKey) error {
	if err := m.persistent.DeleteTeam(userID, teamID); err != nil {
		return err
	}
	if err := m.inmem.DeleteTeam(userID, teamID); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) UpdateTeam(userID model.UserKey, team model.Team) error {
	if err := m.persistent.UpdateTeam(userID, team); err != nil {
		return err
	}
	if err := m.inmem.UpdateTeam(userID, team); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveTeamTags(userID model.UserKey, teamID string) (model.Tags, error) {
	return m.inmem.RetrieveTeamTags(userID, teamID)
}

func (m *memoryDB) UpdateTeamTags(userID model.UserKey, teamID string, tags model.Tags) error {
	if err := m.persistent.UpdateTeamTags(userID, teamID, tags); err != nil {
		return err
	}
	if err := m.inmem.UpdateTeamTags(userID, teamID, tags); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveUserByExternalID(userID string, method model.AuthMethod) (*model.User, error) {
	return m.inmem.RetrieveUserByExternalID(userID, method)
}

func (m *memoryDB) NewUserID() model.UserKey {
	return m.inmem.NewUserID()
}

func (m *memoryDB) CreateUser(user model.User, privateTeam model.Team) error {
	if err := m.persistent.CreateUser(user, privateTeam); err != nil {
		return err
	}
	if err := m.inmem.CreateUser(user, privateTeam); err != nil {
		return err
	}
	return nil
}

func (m *memoryDB) UpdateUser(user *model.User) error {
	if err := m.persistent.UpdateUser(user); err != nil {
		return err
	}
	if err := m.inmem.UpdateUser(user); err != nil {
		return err
	}
	return nil
}

func (m *memoryDB) RetrieveUser(userID model.UserKey) (model.User, error) {
	return m.inmem.RetrieveUser(userID)
}

func (m *memoryDB) CreateAPN(apn model.APN) error {
	if err := m.persistentAPN.CreateAPN(apn); err != nil {
		return err
	}
	if err := m.inmemAPN.CreateAPN(apn); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RemoveAPN(apnID int) error {
	if err := m.persistentAPN.RemoveAPN(apnID); err != nil {
		return err
	}
	if err := m.inmemAPN.RemoveAPN(apnID); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) CreateNAS(nas model.NAS) error {
	if err := m.persistentAPN.CreateNAS(nas); err != nil {
		return err
	}
	if err := m.inmemAPN.CreateNAS(nas); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RemoveNAS(apnID, nasID int) error {
	if err := m.persistentAPN.RemoveNAS(apnID, nasID); err != nil {
		return err
	}
	if err := m.inmemAPN.RemoveNAS(apnID, nasID); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) ListAPN() ([]model.APN, error) {
	return m.inmemAPN.ListAPN()
}

func (m *memoryDB) ListNAS(apnID int) ([]model.NAS, error) {
	return m.inmemAPN.ListNAS(apnID)
}

func (m *memoryDB) ListAllocations(apnID, nasID, maxRows int) ([]model.Allocation, error) {
	return m.inmemAPN.ListAllocations(apnID, nasID, maxRows)
}

func (m *memoryDB) CreateAllocation(allocation model.Allocation) error {
	if err := m.persistentAPN.CreateAllocation(allocation); err != nil {
		return err
	}
	if err := m.inmemAPN.CreateAllocation(allocation); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RemoveAllocation(apnID int, nasID int, imsi int64) error {
	if err := m.persistentAPN.RemoveAllocation(apnID, nasID, imsi); err != nil {
		return err
	}
	if err := m.inmemAPN.RemoveAllocation(apnID, nasID, imsi); err != nil {
		panic(err)
	}
	return nil
}

func (m *memoryDB) RetrieveAllAllocations(imsi int64) ([]model.Allocation, error) {
	return m.inmemAPN.RetrieveAllAllocations(imsi)
}

func (m *memoryDB) RetrieveAllocation(imsi int64, apnid int, nasid int) (model.Allocation, error) {
	return m.inmemAPN.RetrieveAllocation(imsi, apnid, nasid)
}

func (m *memoryDB) LookupIMSIFromIP(ip net.IP, ranges model.NASRanges) (int64, error) {
	return m.inmemAPN.LookupIMSIFromIP(ip, ranges)
}

func (m *memoryDB) RetrieveNAS(apnID int, nasid int) (model.NAS, error) {
	return m.inmemAPN.RetrieveNAS(apnID, nasid)
}
