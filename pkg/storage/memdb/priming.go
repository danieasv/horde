package memdb

import (
	"errors"
	"fmt"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
)

// Function to prime the cache with existing data from a backend store.

// Selected tables that are cached.
var tables = []string{
	"hordeuser", "token", "role", "team", "member",
	"firmware", "collection", "device", "output", "invite",
}
var apnTables = []string{
	"apn", "nas", "nasalloc",
}

// These must be merged into the common store
// APN tables:
// "apn", "nas", "nasalloc",
// Other tables
//	"downstream", - message buffering
//  "ghstate", "ghsession" -- github logins
// "lwm2mclient",  -- FOTA support
// Big Data Tables. This is TBD:
//  * "magpie_data" -- upstream messages
//  * "firmware_image" -- FOTA support (image files, 2-500KB in size)

// PrimeMemoryCache primes the data store with data from a backend store. All
// data is read into memory and a memDB implementation is returned. The existing data
// that is created during schema creation must be removed before it is
// transplanted across.
func PrimeMemoryCache(source storage.DataStore) (storage.DataStore, error) {
	sourceDB := sqlstore.SQLConnection(source)
	if sourceDB == nil {
		return nil, errors.New("not a sql store")
	}

	mdb, err := sqlstore.NewSQLStore("sqlite3", "file::memory:?cache=shared", true, 1, 1)
	if err != nil {
		return nil, err
	}

	// Replace internal lookups with the cached version
	cachedUtils := NewCacheLookup(mdb)
	if cachedUtils == nil {
		panic("Not a memdb")
	}
	if err := sqlstore.SetInternalLookups(source, cachedUtils); err != nil {
		panic(err)
	}

	ret := newCacheDB(source, mdb)

	// The SQL store does not include the APN tables.. yet.

	destDB := sqlstore.SQLConnection(mdb)
	if destDB == nil {
		return nil, errors.New("memdb has no connection")
	}
	for _, name := range tables {
		_, err := destDB.Exec("DELETE FROM " + name)
		if err != nil {
			panic(err)
		}
		logging.Info("Loading data from %s...", name)
		if err := MoveData(name, sourceDB, destDB, ""); err != nil {
			panic(fmt.Sprintf("Unable to move data into memory cache: %v", err))
		}
	}

	return ret, nil
}

// PrimeAPNCache primes the APN cached storage
func PrimeAPNCache(source storage.APNStore) (storage.APNStore, error) {
	sourceDB := sqlstore.SQLAPNConnection(source)
	if sourceDB == nil {
		return nil, errors.New("not a sql store")
	}

	memdb, err := sqlstore.NewSQLAPNStore("sqlite3", "file::memory:?cache=shared", true)
	if err != nil {
		return nil, err
	}

	ret := newAPNCacheDB(source, memdb)

	// The SQL store does not include the APN tables.. yet.

	destDB := sqlstore.SQLAPNConnection(memdb)
	if destDB == nil {
		return nil, errors.New("memdb has no connection")
	}
	for _, name := range apnTables {
		_, err := destDB.Exec("DELETE FROM " + name)
		if err != nil {
			panic(err)
		}
		logging.Info("Loading data from %s...", name)
		if err := MoveData(name, sourceDB, destDB, ""); err != nil {
			panic(fmt.Sprintf("Unable to move data into memory cache: %v", err))
		}
	}
	return ret, nil
}
