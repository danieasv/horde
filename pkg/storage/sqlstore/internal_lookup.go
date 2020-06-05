package sqlstore

import (
	"database/sql"

	"github.com/eesrc/horde/pkg/model"
)

// InternalLookups is an interface for internal database lookups that are
// reads. When using a caching backend these lookups can be done in the cache
// rather than in the database itself.
// There are cases where there will be race conditions and integrity violations
// (f.e. if someone removes an admin from a team while the same member does
// a change on one of the resources the team owns) but this is an acceptable
// tradeoff.
type InternalLookups interface {
	// Prepare the internal lookups for execution. Typically prepare sql
	// statements for execution
	Prepare(db *sql.DB) error

	// EnsureAdminOfTeam ensures that the user is an admin member of the team
	EnsureAdminOfTeam(tx *sql.Tx, userID model.UserKey, teamID model.TeamKey) error

	// EnsureAdminOfCollection ensures that the user is an admin of the team
	// that owns the collection
	EnsureAdminOfCollection(tx *sql.Tx, userID model.UserKey, collectionID model.CollectionKey) (model.TeamKey, error)

	// EnsureAdminOfDevice ensures that the user is an admin of the team that
	// owns the collection the device is a part of.
	EnsureAdminOfDevice(tx *sql.Tx, userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) error

	// EnsureAdminOfOutput ensures that the user is an admin of the team that
	// owns the collection that contains the output.
	EnsureAdminOfOutput(tx *sql.Tx, userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) error

	// EnsureNotPrivateTeam ensures that the team is not a private team
	EnsureNotPrivateTeam(tx *sql.Tx, teamID model.TeamKey) error

	// EsnureAdminOfFirmware ensures that the user is a member of a team that
	// owns the collection that the firmware is a part of.
	EnsureAdminOfFirmware(tx *sql.Tx, userID model.UserKey, fwID model.FirmwareKey) error

	// EnsureCollectionFirmware ensures that the collection contains the firmware
	EnsureCollectionFirmware(tx *sql.Tx, collectionID model.CollectionKey, fwID model.FirmwareKey) error
}
