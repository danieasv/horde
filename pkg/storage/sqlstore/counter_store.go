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

	"github.com/ExploratoryEngineering/logging"
)

// SQLCounterStore is used to retrieve entity counts. The counts are read at creation.
type SQLCounterStore struct {
	db *sql.DB
}

// NewCounterStore creates a CounterStore implemention
func NewCounterStore(dbParams Parameters) (*SQLCounterStore, error) {

	db, err := sql.Open(dbParams.Type, dbParams.ConnectionString)
	if err != nil {
		return nil, err
	}

	return &SQLCounterStore{db: db}, nil
}

func (s *SQLCounterStore) queryCount(query string) int64 {
	if s.db == nil {
		logging.Warning("No DB available to query. Returning 0 (%s)", query)
		return 0
	}

	row := s.db.QueryRow(query)
	if row == nil {
		logging.Warning("Row == nil. Returning 0 (%s)", query)
		return 0
	}
	users := int64(0)
	if err := row.Scan(&users); err != nil {
		logging.Warning("Got error querying for count(query = %s): %v", query, err)
	}
	return users

}

// Users returns the current number of users in the store
func (s *SQLCounterStore) Users() int64 {
	return s.queryCount("SELECT COUNT(user_id) FROM hordeuser")
}

// Collections returns the current number of collections in the store
func (s *SQLCounterStore) Collections() int64 {
	return s.queryCount("SELECT COUNT(collection_id) FROM collection")
}

// Teams returns the current number of teams in the store
func (s *SQLCounterStore) Teams() int64 {
	return s.queryCount("SELECT COUNT(team_id) FROM team")
}

// Devices returns the current number of devices in the store
func (s *SQLCounterStore) Devices() int64 {
	return s.queryCount("SELECT COUNT(device_id) FROM device")
}

// Outputs returns the current number of outputs in the store
func (s *SQLCounterStore) Outputs() int64 {
	return s.queryCount("SELECT COUNT(output_id) FROM output")
}
