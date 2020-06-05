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

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

type firmwareStatements struct {
	insert            *sql.Stmt
	retrieve          *sql.Stmt
	delete            *sql.Stmt
	list              *sql.Stmt
	update            *sql.Stmt
	retrieveTags      *sql.Stmt
	updateTags        *sql.Stmt
	retrieveTwo       *sql.Stmt
	retrieveConfig    *sql.Stmt
	firmwareUse       *sql.Stmt
	retrieveByVersion *sql.Stmt
}

func (s *sqlStore) initFirmwareStatements() error {
	var err error

	if s.firmwareStatements.insert, err = s.db.Prepare(`
		INSERT INTO firmware (firmware_id, filename, version, length, sha256, created, collection_id, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`); err != nil {
		return err
	}

	if s.firmwareStatements.retrieve, err = s.db.Prepare(`
		SELECT
			fw.firmware_id, fw.filename, fw.version, fw.length, fw.sha256,
			fw.created, fw.collection_id, fw.tags
		FROM
			firmware fw,
			collection c,
			member m
		WHERE
			fw.collection_id = c.collection_id AND
			c.team_id = m.team_id AND
			m.user_id = $1 AND
			c.collection_id = $2 AND
			fw.firmware_id = $3

	`); err != nil {
		return err
	}

	if s.firmwareStatements.delete, err = s.db.Prepare(`
		DELETE
			FROM firmware
		WHERE firmware_id = $1
		`); err != nil {
		return err
	}

	if s.firmwareStatements.list, err = s.db.Prepare(`
		SELECT
			fw.firmware_id, fw.filename, fw.version, fw.length, fw.sha256,
			fw.created, fw.collection_id, fw.tags
		FROM
			firmware fw, collection c, member m
		WHERE fw.collection_id = c.collection_id AND
			c.team_id = m.team_id AND
			m.user_id = $1 AND
			c.collection_id = $2
		ORDER BY fw.version
	`); err != nil {
		return err
	}

	if s.firmwareStatements.update, err = s.db.Prepare(`
		UPDATE firmware
			SET version = $1,
				collection_id = $2,
				tags = $3
			WHERE firmware_id = $4
	`); err != nil {
		return err
	}

	if s.firmwareStatements.retrieveTags, err = s.db.Prepare(`
		SELECT fw.tags
		FROM firmware fw, collection c, member m
		WHERE fw.collection_id = c.collection_id AND
			c.team_id = m.team_id AND
			m.user_id = $1 AND
			fw.firmware_id = $2
	`); err != nil {
		return err
	}
	if s.firmwareStatements.updateTags, err = s.db.Prepare(`
		UPDATE firmware
			SET tags = $1
			WHERE firmware_id IN (
				SELECT fw.firmware_id
				FROM firmware fw, collection c, member m
				WHERE fw.firmware_id  = $2 AND
					fw.collection_id = c.collection_id AND
					c.team_id = m.team_id AND
					m.user_id =  $3 AND m.role_id = 1)
	`); err != nil {
		return err
	}
	if s.firmwareStatements.retrieveTwo, err = s.db.Prepare(`
		SELECT
			fw.firmware_id, fw.filename, fw.version, fw.length, fw.sha256,
			fw.created, fw.collection_id, fw.tags
		FROM
			firmware fw, collection c
		WHERE fw.collection_id = c.collection_id AND
			c.collection_id = $1 AND
			(fw.firmware_id = $2 OR fw.firmware_id = $3)
	`); err != nil {
		return err
	}
	if s.firmwareStatements.retrieveConfig, err = s.db.Prepare(`
		SELECT
			c.fw_current_version AS c_current,
			c.fw_target_version AS c_target,
			c.fw_management,
			d.fw_current_version AS d_current,
			d.fw_target_version AS d_target
		FROM
			device d, collection c
		WHERE
			c.collection_id = d.collection_id AND
			c.collection_id = $1 and d.device_id = $2
	`); err != nil {
		return err
	}
	if s.firmwareStatements.firmwareUse, err = s.db.Prepare(`
		SELECT DISTINCT
			d.device_id, d.fw_current_version, d.fw_target_version
		FROM
			device d, collection c, member m
		WHERE
			d.collection_id = c.collection_id AND
			c.team_id = m.team_id AND m.user_id = $1 AND
			d.collection_id = $2 AND
			(d.fw_current_version = $3 OR d.fw_target_version = $4)
	`); err != nil {
		return err
	}
	if s.firmwareStatements.retrieveByVersion, err = s.db.Prepare(`
		SELECT
			fw.firmware_id, fw.filename, fw.version, fw.length, fw.sha256,
			fw.created, fw.collection_id, fw.tags
		FROM
			firmware fw
		WHERE
			fw.collection_id = $1 AND
			fw.version = $2
	`); err != nil {
		return err
	}
	return nil
}

func (s *sqlStore) NewFirmwareID() model.FirmwareKey {
	return model.FirmwareKey(s.firmwareKeyGen.NewID())
}

func (s *sqlStore) CreateFirmware(userID model.UserKey, fw model.Firmware) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := s.utils.EnsureAdminOfCollection(tx, userID, fw.CollectionID); err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Stmt(s.firmwareStatements.insert).Exec(fw.ID, fw.Filename, fw.Version, fw.Length, fw.SHA256, fw.Created, fw.CollectionID, fw.TagMap)
	if err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "constraint") {
			// sqlite reports column names, postgres reports index name
			if strings.Contains(err.Error(), "sha") {
				return storage.ErrSHAAlreadyExists
			}
			logging.Error("Got constraint: %v", err)
			return storage.ErrAlreadyExists
		}
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) RetrieveFirmware(userID model.UserKey, collectionID model.CollectionKey, fwID model.FirmwareKey) (model.Firmware, error) {
	var ret model.Firmware
	if err := s.firmwareStatements.retrieve.QueryRow(userID, collectionID, fwID).Scan(
		&ret.ID, &ret.Filename, &ret.Version, &ret.Length, &ret.SHA256, &ret.Created,
		&ret.CollectionID, &ret.TagMap); err != nil {
		if err == sql.ErrNoRows {
			return model.Firmware{}, storage.ErrNotFound
		}
		return model.Firmware{}, err
	}
	return ret, nil
}

func (s *sqlStore) DeleteFirmware(userID model.UserKey, collectionID model.CollectionKey, fwID model.FirmwareKey) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := s.utils.EnsureAdminOfCollection(tx, userID, collectionID); err != nil {
		tx.Rollback()
		return err
	}

	result, err := tx.Stmt(s.firmwareStatements.delete).Exec(fwID)
	if err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrReference
		}
		return err
	}
	if count, err := result.RowsAffected(); err != nil || count == 0 {
		tx.Rollback()
		return storage.ErrNotFound
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) ListFirmware(userID model.UserKey, collectionID model.CollectionKey) ([]model.Firmware, error) {
	var ret []model.Firmware
	rows, err := s.firmwareStatements.list.Query(userID, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var fw model.Firmware
		if err := rows.Scan(&fw.ID, &fw.Filename, &fw.Version, &fw.Length, &fw.SHA256, &fw.Created, &fw.CollectionID, &fw.TagMap); err != nil {
			if err == sql.ErrNoRows {
				return nil, storage.ErrNotFound
			}
			return nil, err
		}
		ret = append(ret, fw)
	}
	return ret, nil
}

func (s *sqlStore) UpdateFirmware(userID model.UserKey, collectionID model.CollectionKey, fw model.Firmware) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := s.utils.EnsureAdminOfCollection(tx, userID, collectionID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := s.utils.EnsureAdminOfCollection(tx, userID, fw.CollectionID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Stmt(s.firmwareStatements.update).Exec(
		fw.Version, fw.CollectionID, fw.TagMap, fw.ID); err != nil {
		tx.Rollback()
		// We can get constraint errors if the same version is set for more than
		// one image at a time.
		if strings.Contains(err.Error(), "constraint") {
			logging.Debug("Got constraint error updating firmware: %v", err)
			return storage.ErrAlreadyExists
		}
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) RetrieveFirmwareTags(userID model.UserKey, firmwareID string) (model.Tags, error) {
	k, err := model.NewFirmwareKeyFromString(firmwareID)
	if err != nil {
		return model.Tags{}, storage.ErrNotFound
	}
	ret := model.NewTags()
	if err := s.firmwareStatements.retrieveTags.QueryRow(userID, k).Scan(&ret.TagMap); err != nil {
		if err == sql.ErrNoRows {
			return ret, storage.ErrNotFound
		}
		return ret, err
	}
	return ret, nil
}

func (s *sqlStore) UpdateFirmwareTags(userID model.UserKey, firmwareID string, tags model.Tags) error {
	k, err := model.NewFirmwareKeyFromString(firmwareID)
	if err != nil {
		return storage.ErrNotFound
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Stmt(s.firmwareStatements.updateTags).Exec(tags.TagMap, k, userID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if count, err := res.RowsAffected(); err != nil {
		tx.Rollback()
		return err
	} else if count == 0 {
		err := s.utils.EnsureAdminOfFirmware(tx, userID, k)
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) RetrieveCurrentAndTargetFirmware(collectionID model.CollectionKey, firmwareA model.FirmwareKey, firmwareB model.FirmwareKey) (model.Firmware, model.Firmware, error) {
	var fwA, fwB model.Firmware
	rows, err := s.firmwareStatements.retrieveTwo.Query(collectionID, firmwareA, firmwareB)
	if err != nil {
		return fwA, fwB, err
	}
	found := 0
	expected := 2
	if firmwareA == 0 {
		expected--
	}
	if firmwareB == 0 {
		expected--
	}
	defer rows.Close()
	for rows.Next() {
		var fw model.Firmware
		if err := rows.Scan(&fw.ID, &fw.Filename, &fw.Version, &fw.Length, &fw.SHA256, &fw.Created, &fw.CollectionID, &fw.TagMap); err != nil {
			if err == sql.ErrNoRows {
				return fwA, fwB, storage.ErrNotFound
			}
			return fwA, fwB, err
		}
		if fw.ID == firmwareA {
			fwA = fw
			found++
		}
		if fw.ID == firmwareB {
			fwB = fw
			found++
		}
	}
	if found != expected {
		return fwA, fwB, storage.ErrNotFound
	}
	return fwA, fwB, nil
}

func (s *sqlStore) RetrieveFirmwareConfig(collectionID model.CollectionKey, deviceID model.DeviceKey) (model.FirmwareConfig, error) {
	cfg := model.FirmwareConfig{}
	var ccFW, ctFW, dcFW, dtFW sql.NullInt64
	if err := s.firmwareStatements.retrieveConfig.QueryRow(collectionID, deviceID).Scan(
		&ccFW, &ctFW, &cfg.Management, &dcFW, &dtFW); err != nil {
		if err == sql.ErrNoRows {
			return cfg, storage.ErrNotFound
		}
		return cfg, err
	}
	if ccFW.Valid {
		cfg.CollectionCurrentVersion = model.FirmwareKey(ccFW.Int64)
	}
	if ctFW.Valid {
		cfg.CollectionTargetVersion = model.FirmwareKey(ctFW.Int64)
	}
	if dcFW.Valid {
		cfg.DeviceCurrentVersion = model.FirmwareKey(dcFW.Int64)
	}
	if dtFW.Valid {
		cfg.DeviceTargetVersion = model.FirmwareKey(dtFW.Int64)
	}
	return cfg, nil
}

func (s *sqlStore) RetrieveFirmwareVersionsInUse(userID model.UserKey, collectionID model.CollectionKey, firmwareID model.FirmwareKey) (model.FirmwareUse, error) {
	ret := model.FirmwareUse{
		Current:  make([]model.DeviceKey, 0),
		Targeted: make([]model.DeviceKey, 0),
	}

	rows, err := s.firmwareStatements.firmwareUse.Query(userID, collectionID, firmwareID, firmwareID)
	if err != nil {
		return ret, err
	}
	ret.FirmwareID = firmwareID
	defer rows.Close()
	for rows.Next() {
		var did, cid, tid sql.NullInt64
		if err := rows.Scan(&did, &cid, &tid); err != nil {
			return ret, err
		}
		if !did.Valid {
			logging.Warning("Got NULL element for device ID. Skipping")
			continue
		}
		if cid.Valid && model.FirmwareKey(cid.Int64) == firmwareID {
			ret.Current = append(ret.Current, model.DeviceKey(did.Int64))
		}
		if tid.Valid && model.FirmwareKey(tid.Int64) == firmwareID {
			ret.Targeted = append(ret.Targeted, model.DeviceKey(did.Int64))
		}
	}
	return ret, nil
}

func (s *sqlStore) RetrieveFirmwareByVersion(collectionID model.CollectionKey, version string) (model.Firmware, error) {
	var ret model.Firmware
	if err := s.firmwareStatements.retrieveByVersion.QueryRow(collectionID, strings.TrimSpace(version)).Scan(
		&ret.ID, &ret.Filename, &ret.Version, &ret.Length, &ret.SHA256, &ret.Created,
		&ret.CollectionID, &ret.TagMap); err != nil {
		if err == sql.ErrNoRows {
			return model.Firmware{}, storage.ErrNotFound
		}
		return model.Firmware{}, err
	}
	return ret, nil
}
