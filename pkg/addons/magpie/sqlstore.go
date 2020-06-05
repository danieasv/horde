package magpie

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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ExploratoryEngineering/logging"

	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/storage/sqlstore"

	_ "github.com/lib/pq"           // driver for postgres
	_ "github.com/mattn/go-sqlite3" // driver for database
)

type sqlStore struct {
	db     *sql.DB
	mutex  *sync.Mutex
	insert *sql.Stmt
}

func newSQLStore(cfg sqlstore.Parameters) (backendStore, error) {
	ret := sqlStore{mutex: &sync.Mutex{}}
	var err error
	ret.db, err = sql.Open(cfg.Type, cfg.ConnectionString)
	if err != nil {
		return nil, err
	}
	logging.Debug("Creating schema (if necessary)...")

	if err := ret.initDB(cfg.Type); err != nil {
		ret.db.Close()
		return nil, err
	}
	logging.Debug("schema creation complete")
	if err := ret.initStatements(); err != nil {
		return nil, err
	}
	return &ret, nil
}

func (s *sqlStore) initDB(driver string) error {
	schema := sqlstore.NewSchema(driver, sqlstore.DBDataStoreSchema)
	for i, v := range schema.Statements() {
		_, err := s.db.Exec(v)
		if err != nil {
			return fmt.Errorf("unable to execute command #%d %s: %v", i, v, err)
		}
	}
	return nil
}

func (s *sqlStore) initStatements() error {
	var err error
	if s.insert, err = s.db.Prepare(`
		INSERT INTO magpie_data (
			collection_id, device_id,
			created, inserted, metadata,
			payload)
		VALUES ($1, $2, $3, $4, $5, $6)`); err != nil {
		return err
	}
	return nil
}
func (s *sqlStore) Store(msg *datastore.DataMessage) error {
	if msg == nil {
		return errors.New("msg is nil")
	}
	res, err := s.insert.Exec(msg.CollectionId, msg.DeviceId, msg.Created, time.Now().UnixNano(), msg.Metadata, msg.Payload)
	if err != nil {
		return err
	}
	if count, err := res.RowsAffected(); err != nil && count == 0 {
		return errors.New("no row inserted")
	}
	return nil
}

func (s *sqlStore) Query(filter *datastore.DataFilter) (chan datastore.DataMessage, error) {

	if filter.CollectionId == "" {
		return nil, errors.New("must specify collection ID")
	}

	whereClause, args := s.makeFilter(filter)
	query := "SELECT collection_id, device_id, created, metadata, payload FROM magpie_data " + whereClause + " ORDER BY created DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	res, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	result := make(chan datastore.DataMessage)
	go func(rows *sql.Rows, ch chan datastore.DataMessage) {
		defer rows.Close()
		defer close(ch)
		s.mutex.Lock()
		for rows.Next() {
			m := datastore.DataMessage{}
			if err := rows.Scan(&m.CollectionId, &m.DeviceId, &m.Created, &m.Metadata, &m.Payload); err != nil {
				logging.Debug("Error Scan(): %v", err)
				return
			}
			ch <- m
		}
		defer s.mutex.Unlock()
	}(res, result)
	return result, nil
}

func (s *sqlStore) makeFilter(filter *datastore.DataFilter) (string, []interface{}) {
	var args []interface{}

	paramno := 1
	whereClause := fmt.Sprintf("WHERE collection_id = $%d", paramno)
	paramno++
	args = append(args, filter.CollectionId)
	if filter.DeviceId != "" {
		whereClause += fmt.Sprintf(" AND device_id = $%d", paramno)
		paramno++
		args = append(args, filter.DeviceId)
	}
	if filter.From > 0 {
		whereClause += fmt.Sprintf(" AND created >= $%d", paramno)
		paramno++
		args = append(args, filter.From)
	}
	if filter.To > 0 {
		whereClause += fmt.Sprintf(" AND created <= $%d", paramno)
		paramno++
		args = append(args, filter.To)
	}
	return whereClause, args
}

func (s *sqlStore) Metrics(filter *datastore.DataFilter) (datastore.DataMetrics, error) {
	ret := datastore.DataMetrics{}
	whereClause, args := s.makeFilter(filter)
	query := `SELECT MIN(created) AS first, MAX(created) AS last, COUNT(*) AS count FROM magpie_data ` + whereClause

	res, err := s.db.Query(query, args...)
	if err != nil {
		return ret, err
	}

	defer res.Close()
	if !res.Next() {
		return ret, errors.New("no rows returned")
	}

	var first, last, count sql.NullInt64
	err = res.Scan(&first, &last, &count)
	if err != nil {
		return ret, err
	}
	if !first.Valid && !last.Valid && !count.Valid {
		return ret, errors.New("not enough valid values")
	}
	ret.FirstDataPoint = first.Int64
	ret.LastDataPoint = last.Int64
	ret.MessageCount = count.Int64
	return ret, err
}
