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
	"errors"
	"fmt"
	"strings"

	"github.com/ExploratoryEngineering/logging"
	"github.com/TelenorDigital/goconnect"
	"github.com/eesrc/horde/pkg/storage"

	// SQLite3 driver for testing, local instances and in-memory database
	_ "github.com/mattn/go-sqlite3"
	//PostgreSQL driver for production servers and Real Backends (tm)
	_ "github.com/lib/pq"
)

// rowScanner implements Scan - ie read from both sql.Row and sql.Rows. i'm not sure
// why golang doesn't implement this interface
type rowScanner interface {
	Scan(...interface{}) error
}

// sqlStore is a storage backend for SQLite
type sqlStore struct {
	db                   *sql.DB
	userKeyGen           *storage.KeyGenerator
	teamKeyGen           *storage.KeyGenerator
	collKeyGen           *storage.KeyGenerator
	deviceKeyGen         *storage.KeyGenerator
	outputKeyGen         *storage.KeyGenerator
	firmwareKeyGen       *storage.KeyGenerator
	tokenStatements      tokenStatements
	userStatements       userStatements
	inviteStatements     inviteStatements
	teamStatements       teamStatements
	collectionStatements collectionStatements
	deviceStatements     deviceStatements
	sequenceStatements   sequenceStatements
	outputStatements     outputStatements
	utils                InternalLookups
	firmwareStatements   firmwareStatements
}

// SQLConnection returns the internal *sql.DB connection used by the data store.
// This is not thread safe and the connection should be used with care. It will
// return nil if the store isn't a sql data store.
func SQLConnection(store storage.DataStore) *sql.DB {
	s, ok := store.(*sqlStore)
	if !ok {
		return nil
	}
	return s.db
}

// GetInternalLookups returns the current utility lookup implementation. This is not
// thread safe. Returns nil if the store isn't a SQL-backed data store
func GetInternalLookups(store storage.DataStore) InternalLookups {
	s, ok := store.(*sqlStore)
	if !ok {
		return nil
	}
	return s.utils
}

// SetInternalLookups replaces the current utility lookups with another. This is
// not thread safe. Returns an error if the store isn't a SQL-backed data store.
func SetInternalLookups(store storage.DataStore, newLookups InternalLookups) error {
	s, ok := store.(*sqlStore)
	if !ok {
		return errors.New("not a sql-backed store")
	}
	s.utils = newLookups
	return nil
}

// NewSQLStoreWithConnection creates a DataStore instance with an existing sql.DB connection.
func NewSQLStoreWithConnection(db *sql.DB, dataCenterID uint8, workerID uint16) (storage.DataStore, error) {
	ret := &sqlStore{db: db, utils: NewInternalLookup()}

	ret.userKeyGen = storage.NewKeyGenerator(dataCenterID, workerID, "user", ret)
	ret.userKeyGen.Start()

	ret.teamKeyGen = storage.NewKeyGenerator(dataCenterID, workerID, "team", ret)
	ret.teamKeyGen.Start()

	ret.collKeyGen = storage.NewKeyGenerator(dataCenterID, workerID, "coll", ret)
	ret.collKeyGen.Start()

	ret.deviceKeyGen = storage.NewKeyGenerator(dataCenterID, workerID, "dev", ret)
	ret.deviceKeyGen.Start()

	ret.outputKeyGen = storage.NewKeyGenerator(dataCenterID, workerID, "out", ret)
	ret.outputKeyGen.Start()

	ret.firmwareKeyGen = storage.NewKeyGenerator(dataCenterID, workerID, "fw", ret)
	ret.firmwareKeyGen.Start()

	if err := ret.initTokenStatements(); err != nil {
		return nil, fmt.Errorf("error preparing token statements: %v", err)
	}
	if err := ret.initUserStatements(); err != nil {
		return nil, fmt.Errorf("error preparing user statements: %v", err)
	}
	if err := ret.initInviteStatements(); err != nil {
		return nil, fmt.Errorf("error preparing invite statements: %v", err)
	}
	if err := ret.initTeamStatements(); err != nil {
		return nil, fmt.Errorf("error preparing team statements: %v", err)
	}
	if err := ret.initCollectionStratements(); err != nil {
		return nil, fmt.Errorf("error preparing collection statements: %v", err)
	}
	if err := ret.initDeviceStatements(); err != nil {
		return nil, fmt.Errorf("error preparing device statements: %v", err)
	}
	if err := ret.initSequenceStatements(); err != nil {
		return nil, fmt.Errorf("error preparing sequence statements: %v", err)
	}
	if err := ret.initOutputStatements(); err != nil {
		return nil, fmt.Errorf("error preparing output statements: %v", err)
	}
	if err := ret.initFirmwareStatements(); err != nil {
		return nil, fmt.Errorf("error preparing firmware statements: %v", err)
	}

	if err := ret.utils.Prepare(ret.db); err != nil {
		return nil, fmt.Errorf("error preparing util statements: %v", err)
	}

	return ret, nil
}

// NewSQLStore creates a new SequenceStore for a SQL backend
func NewSQLStore(driver string, connectionString string, create bool, dataCenterID uint8, workerID uint16) (storage.DataStore, error) {
	db, err := sql.Open(driver, connectionString)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("unable to ping %s database: %v", driver, err)
	}

	// Always create schema if this is a memory-backed instance
	if create || strings.HasPrefix(connectionString, ":memory:") {
		logging.Info("Creating schema")
		schema := NewSchema(driver, DBSchema, strings.Join(goconnect.SQLSchema, ";\n"))
		if err := schema.Create(db); err != nil {
			return nil, err
		}
	}

	return NewSQLStoreWithConnection(db, dataCenterID, workerID)
}

// NewMemoryStore creates a memory-backed SQLite3 instance. Panics if the
// store can't be created. Use for testing
func NewMemoryStore() storage.DataStore {
	s, err := NewSQLMutexStore("sqlite3", ":memory:", true, 1, 1)
	if err != nil {
		panic(err)
	}
	return s
}

// NewMemoryAPNStore returns a memory-based alloc store
func NewMemoryAPNStore() storage.APNStore {
	s, err := NewSQLAPNStore("sqlite3", ":memory:", true)
	if err != nil {
		panic(err)
	}
	return s
}
