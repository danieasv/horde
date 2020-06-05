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
	"fmt"
	"net"
	"strings"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
)

type sqlAPNStore struct {
	db                 *sql.DB
	createAPN          *sql.Stmt
	deleteAPN          *sql.Stmt
	createNAS          *sql.Stmt
	deleteNAS          *sql.Stmt
	listAPN            *sql.Stmt
	listNAS            *sql.Stmt
	createAllocation   *sql.Stmt
	deleteAllocation   *sql.Stmt
	listAllocation     *sql.Stmt
	listAllAllocations *sql.Stmt
	getAllocation      *sql.Stmt
	lookupIMSI         *sql.Stmt
	getNAS             *sql.Stmt
}

// NewSQLAPNStore creates a new APN store implementation
func NewSQLAPNStore(driver string, connectionString string, create bool) (storage.APNStore, error) {
	db, err := sql.Open(driver, connectionString)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("unable to ping %s database: %v", driver, err)
	}
	if create || strings.HasPrefix(connectionString, ":memory:") {
		schema := NewSchema(driver, DBAPNSchema)
		if err := schema.Create(db); err != nil {
			return nil, err
		}
	}
	ret := &sqlAPNStore{db: db}
	if err := ret.initStatements(); err != nil {
		logging.Warning("Error init statements")
		return nil, err
	}
	return ret, nil
}

// SQLAPNConnection returns the internal *sql.DB connection used by the data store.
// This is not thread safe and the connection should be used with care. It will
// return nil if the store isn't a sql data store.
func SQLAPNConnection(store storage.APNStore) *sql.DB {
	s, ok := store.(*sqlAPNStore)
	if !ok {
		return nil
	}
	return s.db
}

func (s *sqlAPNStore) initStatements() error {
	var err error
	if s.createAPN, err = s.db.Prepare(`
		INSERT INTO apn (apn_id, name) VALUES ($1, $2)`); err != nil {
		return err
	}
	if s.deleteAPN, err = s.db.Prepare(`
		DELETE FROM apn WHERE apn_id = $1`); err != nil {
		return err
	}
	if s.createNAS, err = s.db.Prepare(`
		INSERT INTO nas (nas_id, identifier, cidr, apn_id) VALUES ($1, $2, $3, $4)`); err != nil {
		return err
	}
	if s.deleteNAS, err = s.db.Prepare(`
		DELETE FROM nas WHERE apn_id = $1 AND nas_id = $2`); err != nil {
		return err
	}
	if s.listAPN, err = s.db.Prepare(`
		SELECT apn_id, name FROM apn ORDER BY apn_id`); err != nil {
		return err
	}
	if s.listNAS, err = s.db.Prepare(`
		SELECT nas_id, identifier, cidr, apn_id
		FROM nas
		WHERE apn_id = $1
		ORDER BY nas_id`); err != nil {
		return err
	}
	if s.createAllocation, err = s.db.Prepare(`
		INSERT INTO nasalloc (apn_id, nas_id, imsi, imei, ip, created)
		VALUES ($1, $2, $3 ,$4, $5, $6)`); err != nil {
		return err
	}
	if s.deleteAllocation, err = s.db.Prepare(`
		DELETE FROM nasalloc
			WHERE imsi = $1 AND apn_id = $2 AND nas_id = $3`); err != nil {
		return err
	}
	if s.listAllocation, err = s.db.Prepare(`
		SELECT apn_id, nas_id, imsi, imei, ip, created
			FROM nasalloc
			WHERE apn_id = $1 AND nas_id = $2
	`); err != nil {
		return err
	}
	if s.listAllAllocations, err = s.db.Prepare(`
		SELECT apn_id, nas_id, imsi, imei, ip, created
			FROM nasalloc
			WHERE imsi = $1
	`); err != nil {
		return err
	}
	if s.getAllocation, err = s.db.Prepare(`
		SELECT apn_id, nas_id, imsi, imei, ip, created
			FROM nasalloc
			WHERE imsi = $1 AND apn_id = $2 AND nas_id = $3
	`); err != nil {
		return err
	}
	if s.lookupIMSI, err = s.db.Prepare(`
		SELECT imsi, nas_id
			FROM nasalloc
			WHERE ip = $1 AND apn_id = $2
	`); err != nil {
		return err
	}
	if s.getNAS, err = s.db.Prepare(`
		SELECT nas_id, identifier, cidr, apn_id
			FROM nas
			WHERE apn_id = $1 AND nas_id = $2
	`); err != nil {
		return err
	}

	return nil
}

func (s *sqlAPNStore) CreateAPN(apn model.APN) error {
	_, err := s.createAPN.Exec(apn.ID, apn.Name)
	if err != nil {
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		logging.Warning("Unable to create APN: %v", err)
		return storage.ErrInternal
	}
	return nil
}

func (s *sqlAPNStore) RemoveAPN(apnID int) error {
	res, err := s.deleteAPN.Exec(apnID)
	if err != nil {
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrReference
		}
		logging.Warning("Unable to remove APN: %v", err)
		return storage.ErrInternal
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *sqlAPNStore) CreateNAS(nas model.NAS) error {
	_, err := s.createNAS.Exec(nas.ID, nas.Identifier, nas.CIDR, nas.ApnID)
	if err != nil {
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		logging.Warning("Unable to create NAS: %v", err)
		return storage.ErrInternal
	}
	return nil
}

func (s *sqlAPNStore) RemoveNAS(apnID, nasID int) error {
	res, err := s.deleteNAS.Exec(apnID, nasID)
	if err != nil {
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrReference
		}
		logging.Warning("Unable to remove NAS: %v", err)
		return storage.ErrInternal
	}
	if count, _ := res.RowsAffected(); count == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *sqlAPNStore) ListAPN() ([]model.APN, error) {
	rows, err := s.listAPN.Query()
	var ret []model.APN

	if err != nil {
		if err == sql.ErrNoRows {
			return ret, nil
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var apn model.APN
		if err := rows.Scan(&apn.ID, &apn.Name); err != nil {
			return nil, err
		}
		ret = append(ret, apn)
	}
	return ret, nil
}

func (s *sqlAPNStore) rowsToNAS(r *sql.Rows) (model.NAS, error) {
	var nas model.NAS
	if err := r.Scan(&nas.ID, &nas.Identifier, &nas.CIDR, &nas.ApnID); err != nil {
		return nas, err
	}
	return nas, nil
}
func (s *sqlAPNStore) ListNAS(apnID int) ([]model.NAS, error) {
	rows, err := s.listNAS.Query(apnID)
	var ret []model.NAS

	if err != nil {
		if err == sql.ErrNoRows {
			return ret, nil
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		nas, err := s.rowsToNAS(rows)
		if err != nil {
			return ret, err
		}
		ret = append(ret, nas)
	}
	return ret, nil
}

func (s *sqlAPNStore) CreateAllocation(alloc model.Allocation) error {
	imei := sql.NullInt64{Valid: false}
	if alloc.IMEI > 0 {
		imei.Valid = true
		imei.Int64 = alloc.IMEI
	}
	_, err := s.createAllocation.Exec(alloc.ApnID, alloc.NasID, alloc.IMSI,
		imei, alloc.IP.String(), alloc.Created)
	if err != nil {
		logging.Warning("Unable to create allocation %+v: %v", alloc, err)
		if strings.Contains(err.Error(), "constraint") {
			return storage.ErrAlreadyExists
		}
		return err
	}
	return nil
}
func (s *sqlAPNStore) rowsToAllocation(rows *sql.Rows) (model.Allocation, error) {
	var alloc model.Allocation
	var imei sql.NullInt64
	ip := ""
	if err := rows.Scan(&alloc.ApnID, &alloc.NasID, &alloc.IMSI, &imei, &ip, &alloc.Created); err != nil {
		return alloc, err
	}
	alloc.IP = net.ParseIP(ip)
	alloc.IMEI = 0
	if imei.Valid {
		alloc.IMEI = imei.Int64
	}
	return alloc, nil
}

func (s *sqlAPNStore) readAllocationRows(rows *sql.Rows, err error, maxRows int) ([]model.Allocation, error) {
	var ret []model.Allocation

	if err != nil {
		if err == sql.ErrNoRows {
			return ret, nil
		}
		return nil, err
	}
	defer rows.Close()
	count := 1
	for rows.Next() {
		alloc, err := s.rowsToAllocation(rows)
		if err != nil {
			return nil, err
		}
		ret = append(ret, alloc)
		count = count + 1
		if count > maxRows {
			break
		}
	}
	return ret, nil
}
func (s *sqlAPNStore) ListAllocations(apnID, nasID, maxRows int) ([]model.Allocation, error) {
	rows, err := s.listAllocation.Query(apnID, nasID)
	return s.readAllocationRows(rows, err, maxRows)
}

func (s *sqlAPNStore) RemoveAllocation(apnID int, nasID int, imsi int64) error {
	result, err := s.deleteAllocation.Exec(imsi, apnID, nasID)
	if err != nil {
		logging.Warning("Unable to remove allocation with IMSI %d (APN: %d, NAS: %d): %v", imsi, apnID, nasID, err)
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (s *sqlAPNStore) RetrieveAllAllocations(imsi int64) ([]model.Allocation, error) {
	rows, err := s.listAllAllocations.Query(imsi)
	return s.readAllocationRows(rows, err, 10000)
}

func (s *sqlAPNStore) RetrieveAllocation(imsi int64, apnid int, nasid int) (model.Allocation, error) {
	rows, err := s.getAllocation.Query(imsi, apnid, nasid)
	ret, err := s.readAllocationRows(rows, err, 1)
	if err != nil {
		return model.Allocation{}, err
	}
	if len(ret) == 0 {
		return model.Allocation{}, storage.ErrNotFound
	}
	return ret[0], nil
}

func (s *sqlAPNStore) LookupIMSIFromIP(ip net.IP, ranges model.NASRanges) (int64, error) {
	rows, err := s.lookupIMSI.Query(ip.String(), ranges.APN.ID)
	if err != nil {
		logging.Warning("Unable to do IMSI lookup: %v", err)
		return 0, storage.ErrInternal
	}
	defer rows.Close()
	for rows.Next() {
		var nas sql.NullInt64
		var imsi int64
		if err := rows.Scan(&imsi, &nas); err != nil {
			logging.Warning("Unable to do IMSI lookup: %v", err)
			return 0, storage.ErrInternal
		}
		if nas.Valid {
			for _, n := range ranges.Ranges {
				if n.ID == int(nas.Int64) {
					return imsi, nil
				}
			}
		}
	}
	return 0, storage.ErrNotFound
}

func (s *sqlAPNStore) RetrieveNAS(apnID int, nasID int) (model.NAS, error) {
	rows, err := s.getNAS.Query(apnID, nasID)

	if err != nil {
		return model.NAS{}, err
	}
	defer rows.Close()
	if rows.Next() {
		nas, err := s.rowsToNAS(rows)
		if err != nil {
			return model.NAS{}, err
		}
		return nas, nil
	}
	return model.NAS{}, storage.ErrNotFound
}
