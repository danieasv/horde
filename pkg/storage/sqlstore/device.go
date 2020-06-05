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
	"github.com/lib/pq"
)

func (s *sqlStore) NewDeviceID() model.DeviceKey {
	return model.DeviceKey(s.deviceKeyGen.NewID())
}

type deviceStatements struct {
	create               *sql.Stmt
	list                 *sql.Stmt
	memberCheck          *sql.Stmt
	retrieve             *sql.Stmt
	delete               *sql.Stmt
	update               *sql.Stmt
	retrieveTags         *sql.Stmt
	updateTags           *sql.Stmt
	collectionMembership *sql.Stmt
	retrieveByIMSI       *sql.Stmt
	retrieveByMSISDN     *sql.Stmt
	allocUpdate          *sql.Stmt
	fwStateUpdate        *sql.Stmt
}

func (s *sqlStore) initDeviceStatements() error {
	var err error
	if s.deviceStatements.create, err = s.db.Prepare(`
		INSERT INTO device (
				device_id,
				imsi,
				imei,
				collection_id,
				tags,
				net_apn_id,
				net_nas_id,
				net_allocated_ip,
				net_allocated_at,
				net_cell_id,
				fw_current_version,
				fw_target_version,
				fw_serial_number,
				fw_model_number,
				fw_manufacturer,
				fw_version,
				fw_state,
				fw_state_message)
		VALUES ($1,
				$2,
				$3,
				$4,
				$5,
				$6,
				$7,
				$8,
				$9,
				$10,
				$11,
				$12,
				$13,
				$14,
				$15,
				$16,
				$17,
				$18)
			`); err != nil {
		return err
	}
	if s.deviceStatements.list, err = s.db.Prepare(`
		SELECT
			d.device_id,
			d.imsi,
			d.imei,
			d.collection_id,
			d.tags,
			d.net_apn_id,
			d.net_nas_id,
			d.net_allocated_ip,
			d.net_allocated_at,
			d.net_cell_id,
			d.fw_current_version,
			d.fw_target_version,
			d.fw_serial_number,
			d.fw_model_number,
			d.fw_manufacturer,
			d.fw_version,
			d.fw_state,
			d.fw_state_message
		FROM
			device d, collection c, member m
		WHERE
			d.collection_id = c.collection_id AND
			c.team_id = m.team_id AND
			m.user_id = $1 AND
			c.collection_id = $2
		`); err != nil {
		return err
	}
	// TODO: Might be better to check membership separately
	if s.deviceStatements.memberCheck, err = s.db.Prepare(`
		SELECT
			COUNT(*)
		FROM
			collection c, member m
		WHERE
			c.collection_id = $1 AND
			c.team_id = m.team_id AND
			m.user_id = $2
		`); err != nil {
		return err
	}
	if s.deviceStatements.retrieve, err = s.db.Prepare(`
		SELECT
			d.device_id,
			d.imsi,
			d.imei,
			d.collection_id,
			d.tags,
			d.net_apn_id,
			d.net_nas_id,
			d.net_allocated_ip,
			d.net_allocated_at,
			d.net_cell_id,
			d.fw_current_version,
			d.fw_target_version,
			d.fw_serial_number,
			d.fw_model_number,
			d.fw_manufacturer,
			d.fw_version,
			d.fw_state,
			d.fw_state_message
		FROM
			device d, collection c, member m
		WHERE
			d.collection_id = c.collection_id AND
			c.team_id = m.team_id AND
			m.user_id = $1 AND
			d.device_id = $2 AND
			c.collection_id = $3
		`); err != nil {
		return err
	}
	if s.deviceStatements.delete, err = s.db.Prepare(`
		DELETE FROM
			device
		WHERE
			device_id = $1
		`); err != nil {
		return err
	}
	if s.deviceStatements.update, err = s.db.Prepare(`
		UPDATE
			device
		SET
			imsi = $1,
			imei = $2,
			collection_id = $3,
			tags = $4,
			net_apn_id = $5,
			net_nas_id = $6,
			net_allocated_ip = $7,
			net_allocated_at = $8,
			net_cell_id = $9,
			fw_current_version = $10,
			fw_target_version = $11,
			fw_serial_number = $12,
			fw_model_number = $13,
			fw_manufacturer = $14,
			fw_version = $15,
			fw_state = $16,
			fw_state_message = $17
		WHERE
			device_id = $18
		`); err != nil {
		return err
	}
	if s.deviceStatements.retrieveTags, err = s.db.Prepare(`
		SELECT
			d.tags
		FROM
			device d, collection c, member m
		WHERE
			d.device_id = $1 AND
			d.collection_id = c.collection_id AND
			c.team_id = m.team_id AND
			m.user_id = $2
		`); err != nil {
		return err
	}
	if s.deviceStatements.updateTags, err = s.db.Prepare(`
		UPDATE
			device
		SET
			tags = $1
		WHERE
			device_id IN (
				SELECT
					d.device_id
				FROM
					device d, collection c, member m
				WHERE
					d.device_id = $2 AND
					d.collection_id = c.collection_id AND
					c.team_id = m.team_id AND
					user_id = $3 AND
					role_id = 1
			)
		`); err != nil {
		return err
	}
	if s.deviceStatements.collectionMembership, err = s.db.Prepare(`
		SELECT
			d.collection_id
		FROM
			device d
		WHERE
			d.device_id = $1
		`); err != nil {
		return err
	}
	if s.deviceStatements.retrieveByIMSI, err = s.db.Prepare(`
		SELECT
			d.device_id,
			d.imsi,
			d.imei,
			d.collection_id,
			d.tags,
			d.net_apn_id,
			d.net_nas_id,
			d.net_allocated_ip,
			d.net_allocated_at,
			d.net_cell_id,
			d.fw_current_version,
			d.fw_target_version,
			d.fw_serial_number,
			d.fw_model_number,
			d.fw_manufacturer,
			d.fw_version,
			d.fw_state,
			d.fw_state_message
		FROM
			device d
		WHERE
			d.imsi = $1
		`); err != nil {
		return err
	}

	if s.deviceStatements.retrieveByMSISDN, err = s.db.Prepare(`
		SELECT
			d.device_id,
			d.imsi,
			d.imei,
			d.collection_id,
			d.tags,
			d.net_apn_id,
			d.net_nas_id,
			d.net_allocated_ip,
			d.net_allocated_at,
			d.net_cell_id,
			d.fw_current_version,
			d.fw_target_version,
			d.fw_serial_number,
			d.fw_model_number,
			d.fw_manufacturer,
			d.fw_version,
			d.fw_state,
			d.fw_state_message
		FROM
			device d, device_lookup l
		WHERE
			d.imsi = l.imsi AND
			l.msisdn = $1
		`); err != nil {
		return err
	}
	if s.deviceStatements.allocUpdate, err = s.db.Prepare(`
		UPDATE
			device
		SET
			tags = $1,
			net_apn_id = $2,
			net_nas_id = $3,
			net_allocated_ip = $4,
			net_allocated_at = $5,
			net_cell_id = $6,
			fw_current_version = $7,
			fw_target_version = $8,
			fw_serial_number = $9,
			fw_model_number = $10,
			fw_manufacturer = $11,
			fw_version = $12,
			fw_state = $13,
			fw_state_message = $14
		WHERE
			device_id = $15
		`); err != nil {
		return err
	}
	if s.deviceStatements.fwStateUpdate, err = s.db.Prepare(`
		UPDATE
			device
		SET
			fw_state = $1,
			fw_state_message = $2
		WHERE
			imsi = $3
	`); err != nil {
		return err
	}
	return nil
}

func (s *sqlStore) CreateDevice(userID model.UserKey, newDevice model.Device) error {
	tx, err := s.db.Begin()
	if err != nil {
		logging.Warning("Unable to create new transaction: %v", err)
		return err
	}

	if _, err := s.utils.EnsureAdminOfCollection(tx, userID, newDevice.CollectionID); err != nil {
		tx.Rollback()
		return err
	}

	var curVer, tarVer, ci sql.NullInt64
	var ip, sn, mn, mf, fv sql.NullString
	var aa pq.NullTime
	if newDevice.Firmware.CurrentFirmwareID != 0 {
		curVer.Int64 = int64(newDevice.Firmware.CurrentFirmwareID)
		curVer.Valid = true
	}
	if newDevice.Firmware.TargetFirmwareID != 0 {
		tarVer.Int64 = int64(newDevice.Firmware.TargetFirmwareID)
		tarVer.Valid = true
	}
	if newDevice.Network.AllocatedIP != "" {
		ip.String = newDevice.Network.AllocatedIP
		ip.Valid = true
	}
	if newDevice.Firmware.SerialNumber != "" {
		sn.String = newDevice.Firmware.SerialNumber
		sn.Valid = true
	}
	if newDevice.Firmware.ModelNumber != "" {
		mn.String = newDevice.Firmware.ModelNumber
		mn.Valid = true
	}
	if newDevice.Firmware.Manufacturer != "" {
		mf.String = newDevice.Firmware.Manufacturer
		mf.Valid = true
	}
	if newDevice.Firmware.FirmwareVersion != "" {
		fv.String = newDevice.Firmware.FirmwareVersion
		fv.Valid = true
	}
	if newDevice.Network.CellID != 0 {
		ci.Int64 = newDevice.Network.CellID
		ci.Valid = true
	}
	if newDevice.Network.AllocatedAt.Unix() > 0 {
		aa.Time = newDevice.Network.AllocatedAt
		aa.Valid = true
	}
	_, err = tx.Stmt(s.deviceStatements.create).Exec(
		newDevice.ID, newDevice.IMSI, newDevice.IMEI,
		newDevice.CollectionID, newDevice.TagMap, newDevice.Network.ApnID, newDevice.Network.NasID,
		ip, aa, ci, curVer, tarVer, sn, mn, mf, fv, string(newDevice.Firmware.State), newDevice.Firmware.StateMessage)
	if err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) readDevice(row rowScanner) (model.Device, error) {
	var ret model.Device
	var apnID, nasID, curVer, tarVer, ci sql.NullInt64
	var ip, sn, mn, mf, fv sql.NullString
	var aa pq.NullTime
	var stateStr string
	if err := row.Scan(
		&ret.ID, &ret.IMSI, &ret.IMEI, &ret.CollectionID, &ret.TagMap,
		&apnID, &nasID, &ip, &aa, &ci, &curVer, &tarVer, &sn, &mn, &mf, &fv,
		&stateStr, &ret.Firmware.StateMessage); err != nil {
		if err == sql.ErrNoRows {
			return ret, storage.ErrNotFound
		}
		return ret, err
	}
	ret.Firmware.State = model.DeviceFirmwareState(stateStr[0])
	if apnID.Valid {
		ret.Network.ApnID = int(apnID.Int64)
	}
	if nasID.Valid {
		ret.Network.NasID = int(nasID.Int64)
	}
	if curVer.Valid {
		ret.Firmware.CurrentFirmwareID = model.FirmwareKey(curVer.Int64)
	}
	if tarVer.Valid {
		ret.Firmware.TargetFirmwareID = model.FirmwareKey(tarVer.Int64)
	}
	if ip.Valid {
		ret.Network.AllocatedIP = ip.String
	}
	if sn.Valid {
		ret.Firmware.SerialNumber = sn.String
	}
	if mn.Valid {
		ret.Firmware.ModelNumber = mn.String
	}
	if mf.Valid {
		ret.Firmware.Manufacturer = mf.String
	}
	if aa.Valid {
		ret.Network.AllocatedAt = aa.Time
	}
	if ci.Valid {
		ret.Network.CellID = ci.Int64
	}
	if fv.Valid {
		ret.Firmware.FirmwareVersion = fv.String
	}
	return ret, nil
}

func (s *sqlStore) ListDevices(userID model.UserKey, collectionID model.CollectionKey) ([]model.Device, error) {
	var devices []model.Device
	rows, err := s.deviceStatements.list.Query(userID, collectionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		dev, err := s.readDevice(rows)
		if err != nil {
			return devices, err
		}
		devices = append(devices, dev)
	}
	if len(devices) == 0 {
		// This might be either an empty collection. Check if the user is really a member of the collection
		count := 0
		if err := s.deviceStatements.memberCheck.QueryRow(collectionID, userID).Scan(&count); err == sql.ErrNoRows || count == 0 {
			return nil, storage.ErrNotFound
		}
	}
	return devices, nil
}

func (s *sqlStore) RetrieveDevice(userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) (model.Device, error) {
	return s.readDevice(s.deviceStatements.retrieve.QueryRow(userID, deviceID, collectionID))
}

func (s *sqlStore) DeleteDevice(userID model.UserKey, collectionID model.CollectionKey, deviceID model.DeviceKey) error {
	tx, err := s.db.Begin()
	if err != nil {
		logging.Warning("Unable to create new transaction: %v", err)
		return err
	}

	if err := s.utils.EnsureAdminOfDevice(tx, userID, collectionID, deviceID); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Stmt(s.deviceStatements.delete).Exec(deviceID); err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrReference
		}
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) UpdateDevice(userID model.UserKey, collectionID model.CollectionKey, device model.Device) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := s.utils.EnsureAdminOfDevice(tx, userID, collectionID, device.ID); err != nil {
		tx.Rollback()
		return err
	}
	if device.CollectionID != collectionID {
		if _, err := s.utils.EnsureAdminOfCollection(tx, userID, device.CollectionID); err != nil {
			tx.Rollback()
			return err
		}
	}

	var curVer, tarVer, ci sql.NullInt64
	if device.Firmware.CurrentFirmwareID != 0 {
		curVer.Int64 = int64(device.Firmware.CurrentFirmwareID)
		curVer.Valid = true
	}
	if device.Firmware.TargetFirmwareID != 0 {
		tarVer.Int64 = int64(device.Firmware.TargetFirmwareID)
		tarVer.Valid = true
	}
	if device.Network.CellID != 0 {
		ci.Int64 = device.Network.CellID
		ci.Valid = true
	}

	var ip, sn, mn, mf, fv sql.NullString
	if device.Network.AllocatedIP != "" {
		ip.String = device.Network.AllocatedIP
		ip.Valid = true
	}
	if device.Firmware.SerialNumber != "" {
		sn.String = device.Firmware.SerialNumber
		sn.Valid = true
	}
	if device.Firmware.ModelNumber != "" {
		mn.String = device.Firmware.ModelNumber
		mn.Valid = true
	}
	if device.Firmware.Manufacturer != "" {
		mf.String = device.Firmware.Manufacturer
		mf.Valid = true
	}
	if device.Firmware.FirmwareVersion != "" {
		fv.String = device.Firmware.FirmwareVersion
		fv.Valid = true
	}
	var aa pq.NullTime
	if device.Network.AllocatedAt.Unix() > 0 {
		aa.Time = device.Network.AllocatedAt
		aa.Valid = true
	}

	if _, err := tx.Stmt(s.deviceStatements.update).Exec(
		device.IMSI, device.IMEI, device.CollectionID, device.TagMap,
		device.Network.ApnID, device.Network.NasID, ip, aa,
		ci, curVer, tarVer, sn, mn, mf, fv,
		string(device.Firmware.State), device.Firmware.StateMessage, device.ID); err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) RetrieveDeviceTags(userID model.UserKey, deviceID string) (model.Tags, error) {
	k, err := model.NewDeviceKeyFromString(deviceID)
	if err != nil {
		return model.Tags{}, storage.ErrNotFound
	}
	ret := model.NewTags()
	if err = s.deviceStatements.retrieveTags.QueryRow(k, userID).Scan(&ret.TagMap); err != nil {
		if err == sql.ErrNoRows {
			return ret, storage.ErrNotFound
		}
		return ret, err
	}

	return ret, nil
}

func (s *sqlStore) UpdateDeviceTags(userID model.UserKey, deviceID string, tags model.Tags) error {
	k, err := model.NewDeviceKeyFromString(deviceID)
	if err != nil {
		return storage.ErrNotFound
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	res, err := tx.Stmt(s.deviceStatements.updateTags).Exec(tags.TagMap, k, userID)
	if err != nil {
		tx.Rollback()
		return err
	}
	if count, err := res.RowsAffected(); err != nil {
		tx.Rollback()
		return err
	} else if count == 0 {
		var collectionID model.CollectionKey
		if err := tx.Stmt(s.deviceStatements.collectionMembership).QueryRow(k).Scan(&collectionID); err != nil {
			tx.Rollback()
			if err == sql.ErrNoRows {
				return storage.ErrNotFound
			}
			return err
		}
		_, err := s.utils.EnsureAdminOfCollection(tx, userID, collectionID)
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (s *sqlStore) RetrieveDeviceByIMSI(imsi int64) (model.Device, error) {
	return s.readDevice(s.deviceStatements.retrieveByIMSI.QueryRow(imsi))
}

func (s *sqlStore) RetrieveDeviceByMSISDN(msisdn string) (model.Device, error) {
	return s.readDevice(s.deviceStatements.retrieveByMSISDN.QueryRow(msisdn))
}

func (s *sqlStore) UpdateDeviceMetadata(device model.Device) error {
	var curVer, tarVer, ci sql.NullInt64
	if device.Firmware.CurrentFirmwareID != 0 {
		curVer.Int64 = int64(device.Firmware.CurrentFirmwareID)
		curVer.Valid = true
	}
	if device.Firmware.TargetFirmwareID != 0 {
		tarVer.Int64 = int64(device.Firmware.TargetFirmwareID)
		tarVer.Valid = true
	}
	if device.Network.CellID != 0 {
		ci.Int64 = device.Network.CellID
		ci.Valid = true
	}

	var ip, sn, mn, mf, fv sql.NullString
	if device.Network.AllocatedIP != "" {
		ip.String = device.Network.AllocatedIP
		ip.Valid = true
	}
	if device.Firmware.SerialNumber != "" {
		sn.String = device.Firmware.SerialNumber
		sn.Valid = true
	}
	if device.Firmware.ModelNumber != "" {
		mn.String = device.Firmware.ModelNumber
		mn.Valid = true
	}
	if device.Firmware.Manufacturer != "" {
		mf.String = device.Firmware.Manufacturer
		mf.Valid = true
	}
	if device.Firmware.FirmwareVersion != "" {
		fv.String = device.Firmware.FirmwareVersion
		fv.Valid = true
	}
	var aa pq.NullTime
	if device.Network.AllocatedAt.Unix() > 0 {
		aa.Time = device.Network.AllocatedAt
		aa.Valid = true
	}
	res, err := s.deviceStatements.allocUpdate.Exec(
		device.TagMap, device.Network.ApnID, device.Network.NasID, ip,
		aa, ci, curVer, tarVer, sn, mn, mf, fv,
		string(device.Firmware.State), device.Firmware.StateMessage, device.ID)
	if err != nil {
		return err
	}
	if count, err := res.RowsAffected(); err != nil || count == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *sqlStore) UpdateFirmwareStateForDevice(imsi int64, state model.DeviceFirmwareState, message string) error {
	res, err := s.deviceStatements.fwStateUpdate.Exec(string(state), message, imsi)
	if err != nil {
		logging.Warning("Unable to update device firmware metadata for device with IMSI: %d: %v", imsi, err)
	}
	if count, err := res.RowsAffected(); err != nil || count == 0 {
		return storage.ErrNotFound
	}
	return nil
}
