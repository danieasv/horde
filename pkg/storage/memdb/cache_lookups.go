package memdb

import (
	"database/sql"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
)

// NewCacheLookup creates an internal lookup implementation that uses the local
// cached database rather than an external one. This creates dummy transactions
// for the local *sql.DB instance and calls the internal utility lookups
// directly. It's a bit convoluted and
func NewCacheLookup(store storage.DataStore) sqlstore.InternalLookups {
	ret := &cacheLookups{
		db:    sqlstore.SQLConnection(store),
		utils: sqlstore.GetInternalLookups(store),
	}
	if ret.db == nil || ret.utils == nil {
		return nil
	}
	return ret
}

type cacheLookups struct {
	db    *sql.DB
	utils sqlstore.InternalLookups
}

func (c *cacheLookups) Prepare(db *sql.DB) error {
	panic("Can't prepare this")
}

func (c *cacheLookups) EnsureAdminOfTeam(tx *sql.Tx, userID model.UserKey, teamID model.TeamKey) error {
	wtx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer wtx.Commit()
	return c.utils.EnsureAdminOfTeam(wtx, userID, teamID)
}

func (c *cacheLookups) EnsureAdminOfCollection(tx *sql.Tx, userID model.UserKey, collectionID model.CollectionKey) (model.TeamKey, error) {
	wtx, err := c.db.Begin()
	if err != nil {
		return model.TeamKey(0), err
	}
	defer wtx.Commit()
	return c.utils.EnsureAdminOfCollection(wtx, userID, collectionID)
}

func (c *cacheLookups) EnsureAdminOfDevice(tx *sql.Tx, userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) error {
	wtx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer wtx.Commit()
	return c.utils.EnsureAdminOfDevice(wtx, userID, collectionID, deviceID)
}

func (c *cacheLookups) EnsureAdminOfOutput(tx *sql.Tx, userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) error {
	wtx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer wtx.Commit()
	return c.utils.EnsureAdminOfOutput(wtx, userID, collectionID, outputID)
}

func (c *cacheLookups) EnsureNotPrivateTeam(tx *sql.Tx, teamID model.TeamKey) error {
	wtx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer wtx.Commit()
	return c.utils.EnsureNotPrivateTeam(wtx, teamID)
}

func (c *cacheLookups) EnsureAdminOfFirmware(tx *sql.Tx, userID model.UserKey, fwID model.FirmwareKey) error {
	wtx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer wtx.Commit()
	return c.utils.EnsureAdminOfFirmware(wtx, userID, fwID)
}

func (c *cacheLookups) EnsureCollectionFirmware(tx *sql.Tx, collectionID model.CollectionKey, fwID model.FirmwareKey) error {
	wtx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer wtx.Commit()
	return c.utils.EnsureCollectionFirmware(wtx, collectionID, fwID)
}
