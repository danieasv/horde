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
	"strings"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

func (s *sqlStore) NewCollectionID() model.CollectionKey {
	return model.CollectionKey(s.collKeyGen.NewID())
}

type collectionStatements struct {
	list        *sql.Stmt
	create      *sql.Stmt
	retrieve    *sql.Stmt
	update      *sql.Stmt
	delete      *sql.Stmt
	updateTag   *sql.Stmt
	retrieveTag *sql.Stmt
}

func (s *sqlStore) initCollectionStratements() error {
	var err error
	if s.collectionStatements.list, err = s.db.Prepare(`
		SELECT c.collection_id, c.team_id, c.tags, c.field_mask, c.fw_current_version, c.fw_target_version, c.fw_management
			FROM collection c, member m
			WHERE c.team_id = m.team_id AND
				m.user_id = $1`); err != nil {
		return err
	}
	if s.collectionStatements.create, err = s.db.Prepare(`
		INSERT INTO collection (collection_id, team_id, tags, field_mask, fw_current_version, fw_target_version, fw_management)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`); err != nil {
		return err
	}
	if s.collectionStatements.retrieve, err = s.db.Prepare(`
		SELECT c.collection_id, c.team_id, c.tags, c.field_mask, c.fw_current_version, c.fw_target_version, c.fw_management
			FROM collection c, member m
			WHERE c.collection_id = $1 AND
				c.team_id = m.team_id AND
				m.user_id = $2`); err != nil {
		return err
	}
	if s.collectionStatements.update, err = s.db.Prepare(`
		UPDATE collection
			SET team_id = $1,
				tags = $2,
				field_mask = $3,
				fw_current_version = $4,
				fw_target_version = $5,
				fw_management = $6
			WHERE collection_id = $7`); err != nil {
		return err
	}
	if s.collectionStatements.delete, err = s.db.Prepare(`
		DELETE FROM collection
			WHERE collection_id = $1`); err != nil {
		return err
	}
	if s.collectionStatements.retrieveTag, err = s.db.Prepare(`
		SELECT tags
			FROM collection c, member m
			WHERE c.collection_id = $1 AND
				c.team_id = m.team_id AND
				m.user_id = $2
		`); err != nil {
		return err
	}
	if s.collectionStatements.updateTag, err = s.db.Prepare(`
		UPDATE collection
			SET tags = $1
			WHERE collection_id IN (
				SELECT c.collection_id
				FROM collection c
					INNER JOIN member m ON c.team_id = m.team_id
					WHERE c.collection_id = $2 AND
						m.user_id = $3 AND
						m.role_id = 1)
		`); err != nil {
		return err
	}
	return nil
}

func (s *sqlStore) getFirmwareValues(c *model.Collection) (sql.NullInt64, sql.NullInt64) {
	var current, target sql.NullInt64
	if c.Firmware.CurrentFirmwareID != 0 {
		current.Int64 = int64(c.Firmware.CurrentFirmwareID)
		current.Valid = true
	}
	if c.Firmware.TargetFirmwareID != 0 {
		target.Int64 = int64(c.Firmware.TargetFirmwareID)
		target.Valid = true
	}
	return current, target
}

func (s *sqlStore) setFirmwareValues(current, target sql.NullInt64, c *model.Collection) {
	if current.Valid {
		c.Firmware.CurrentFirmwareID = model.FirmwareKey(current.Int64)
	}
	if target.Valid {
		c.Firmware.TargetFirmwareID = model.FirmwareKey(target.Int64)
	}
}

func (s *sqlStore) ListCollections(userID model.UserKey) ([]model.Collection, error) {
	cl := []model.Collection{}
	rows, err := s.collectionStatements.list.Query(userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var coll model.Collection
		var current, target sql.NullInt64
		if err := rows.Scan(
			&coll.ID, &coll.TeamID, &coll.TagMap, &coll.FieldMask,
			&current, &target,
			&coll.Firmware.Management); err != nil {
			return nil, err
		}
		s.setFirmwareValues(current, target, &coll)
		cl = append(cl, coll)
	}
	return cl, nil
}

func (s *sqlStore) CreateCollection(userID model.UserKey, collection model.Collection) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := s.utils.EnsureAdminOfTeam(tx, userID, collection.TeamID); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.utils.EnsureCollectionFirmware(tx, collection.ID, collection.Firmware.CurrentFirmwareID); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.utils.EnsureCollectionFirmware(tx, collection.ID, collection.Firmware.TargetFirmwareID); err != nil {
		tx.Rollback()
		return err
	}

	current, target := s.getFirmwareValues(&collection)
	if _, err = tx.Stmt(s.collectionStatements.create).Exec(
		collection.ID, collection.TeamID, collection.TagMap, collection.FieldMask,
		current, target, collection.Firmware.Management); err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) RetrieveCollection(userID model.UserKey, collectionID model.CollectionKey) (model.Collection, error) {
	ret := model.Collection{}
	var current, target sql.NullInt64
	if err := s.collectionStatements.retrieve.QueryRow(collectionID, userID).Scan(
		&ret.ID, &ret.TeamID, &ret.TagMap, &ret.FieldMask,
		&current, &target, &ret.Firmware.Management); err != nil {
		if err == sql.ErrNoRows {
			return model.Collection{}, storage.ErrNotFound
		}
		return model.Collection{}, err
	}
	s.setFirmwareValues(current, target, &ret)
	return ret, nil
}

func (s *sqlStore) UpdateCollection(userID model.UserKey, collection model.Collection) error {
	// User must be an admin of current team -- ie retrieve current team ID. The result is clunky
	// since we must do multiple queries for a team update
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	// User must be an admin of old team
	currentTeamID, err := s.utils.EnsureAdminOfCollection(tx, userID, collection.ID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if currentTeamID != collection.TeamID {
		// ...and of new team
		if err := s.utils.EnsureAdminOfTeam(tx, userID, collection.TeamID); err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := s.utils.EnsureCollectionFirmware(tx, collection.ID, collection.Firmware.CurrentFirmwareID); err != nil {
		tx.Rollback()
		return err
	}
	if err := s.utils.EnsureCollectionFirmware(tx, collection.ID, collection.Firmware.TargetFirmwareID); err != nil {
		tx.Rollback()
		return err
	}

	current, target := s.getFirmwareValues(&collection)
	_, err = tx.Stmt(s.collectionStatements.update).Exec(
		collection.TeamID, collection.TagMap, collection.FieldMask,
		current, target, collection.Firmware.Management,
		collection.ID)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) DeleteCollection(userID model.UserKey, collectionID model.CollectionKey) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := s.utils.EnsureAdminOfCollection(tx, userID, collectionID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Stmt(s.collectionStatements.delete).Exec(collectionID); err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrReference
		}
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) RetrieveCollectionTags(userID model.UserKey, collectionID string) (model.Tags, error) {
	k, err := model.NewCollectionKeyFromString(collectionID)
	if err != nil {
		return model.Tags{}, storage.ErrNotFound
	}
	ret := model.NewTags()
	if err := s.collectionStatements.retrieveTag.QueryRow(k, userID).Scan(&ret.TagMap); err != nil {
		if err == sql.ErrNoRows {
			return ret, storage.ErrNotFound
		}
		return ret, err
	}
	return ret, nil
}

func (s *sqlStore) UpdateCollectionTags(userID model.UserKey, collectionID string, tags model.Tags) error {
	k, err := model.NewCollectionKeyFromString(collectionID)
	if err != nil {
		return storage.ErrNotFound
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Stmt(s.collectionStatements.updateTag).Exec(tags.TagMap, k, userID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if count, err := res.RowsAffected(); err != nil {
		tx.Rollback()
		return err
	} else if count == 0 {
		_, err := s.utils.EnsureAdminOfCollection(tx, userID, k)
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}
