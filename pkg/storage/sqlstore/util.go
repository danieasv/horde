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
	"database/sql"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

// NewInternalLookup creates a SQL-backed utility lookup
func NewInternalLookup() InternalLookups {
	return &utilStatements{}
}

type utilStatements struct {
	adminOfTeam        *sql.Stmt
	adminOfCollection  *sql.Stmt
	adminOfDevice      *sql.Stmt
	adminOfOutput      *sql.Stmt
	ensureNotPrivate   *sql.Stmt
	adminOfFirmware    *sql.Stmt
	collectionFirmware *sql.Stmt
}

func (u *utilStatements) Prepare(db *sql.DB) error {
	var err error
	if u.adminOfTeam, err = db.Prepare(`
		SELECT m.role_id
			FROM member m
			WHERE m.team_id = $1 AND
				m.user_id = $2`); err != nil {
		return err
	}
	if u.adminOfCollection, err = db.Prepare(`
		SELECT c.team_id, m.role_id
			FROM collection c, member m
			WHERE c.team_id = m.team_id AND
				c.collection_id = $1 AND
				m.user_id = $2`); err != nil {
		return err
	}
	if u.adminOfDevice, err = db.Prepare(`
		SELECT m.role_id
			FROM device d, collection c, member m
			WHERE d.collection_id = c.collection_id AND
				c.team_id = m.team_id AND
				d.device_id = $1 AND
				c.collection_id = $2 AND
				m.user_id = $3`); err != nil {
		return err
	}
	if u.adminOfOutput, err = db.Prepare(`
		SELECT m.role_id
			FROM output d, collection c, member m
			WHERE d.collection_id = c.collection_id AND
				c.team_id = m.team_id AND
				d.output_id = $1 AND
				c.collection_id = $2 AND
				m.user_id = $3`); err != nil {
		return err
	}
	if u.ensureNotPrivate, err = db.Prepare(`
		SELECT t.team_id
			FROM team t, hordeuser u
			WHERE t.team_id = $1 AND
				u.private_team_id = t.team_id`); err != nil {
		return err
	}
	if u.adminOfFirmware, err = db.Prepare(`
		SELECT m.role_id
			FROM firmware fw, collection c, member m
			WHERE fw.collection_id = c.collection_id AND
				c.team_id = m.team_id AND
				fw.firmware_id = $1 AND
				m.user_id = $2
	`); err != nil {
		return err
	}
	if u.collectionFirmware, err = db.Prepare(`
		SELECT collection_id FROM firmware WHERE firmware_id = $1
	`); err != nil {
		return err
	}
	return nil
}

func (u *utilStatements) EnsureAdminOfTeam(tx *sql.Tx, userID model.UserKey, teamID model.TeamKey) error {
	var role model.RoleID
	if err := tx.Stmt(u.adminOfTeam).QueryRow(teamID, userID).Scan(&role); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return storage.ErrNotFound
		}
		return err
	}
	if role != model.AdminRole {
		tx.Rollback()
		return storage.ErrAccess
	}
	return nil
}

func (u *utilStatements) EnsureAdminOfCollection(tx *sql.Tx, userID model.UserKey, collectionID model.CollectionKey) (model.TeamKey, error) {
	var teamID model.TeamKey
	var role model.RoleID
	if err := tx.Stmt(u.adminOfCollection).QueryRow(collectionID, userID).Scan(&teamID, &role); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return 0, storage.ErrNotFound
		}
		return 0, err
	}
	if role != model.AdminRole {
		return teamID, storage.ErrAccess
	}
	return teamID, nil
}

func (u *utilStatements) EnsureAdminOfDevice(tx *sql.Tx, userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) error {
	var role model.RoleID
	if err := tx.Stmt(u.adminOfDevice).QueryRow(deviceID, collectionID, userID).Scan(&role); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return storage.ErrNotFound
		}
		return err
	}
	if role != model.AdminRole {
		return storage.ErrAccess
	}
	return nil
}

func (u *utilStatements) EnsureAdminOfOutput(tx *sql.Tx, userID model.UserKey, collectionID model.CollectionKey, outputID model.OutputKey) error {
	var role model.RoleID
	if err := tx.Stmt(u.adminOfOutput).QueryRow(outputID, collectionID, userID).Scan(&role); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return storage.ErrNotFound
		}
		return err
	}
	if role != model.AdminRole {
		return storage.ErrAccess
	}
	return nil
}

func (u *utilStatements) EnsureNotPrivateTeam(tx *sql.Tx, teamID model.TeamKey) error {
	count := 0
	var err error
	if err = tx.Stmt(u.ensureNotPrivate).QueryRow(teamID).Scan(&count); err != sql.ErrNoRows || count > 0 {
		return storage.ErrAccess
	}

	return nil
}

func (u *utilStatements) EnsureAdminOfFirmware(tx *sql.Tx, userID model.UserKey, fwID model.FirmwareKey) error {
	var role model.RoleID
	if err := tx.Stmt(u.adminOfFirmware).QueryRow(fwID, userID).Scan(&role); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return storage.ErrNotFound
		}
		return err
	}
	if role != model.AdminRole {
		return storage.ErrAccess
	}
	return nil
}

func (u *utilStatements) EnsureCollectionFirmware(tx *sql.Tx, collectionID model.CollectionKey, fwID model.FirmwareKey) error {
	if fwID == 0 {
		return nil
	}
	var cID model.CollectionKey
	if err := tx.Stmt(u.collectionFirmware).QueryRow(fwID).Scan(&cID); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return storage.ErrNotFound
		}
		return err
	}
	if cID != collectionID {
		return storage.ErrAccess
	}
	return nil
}
