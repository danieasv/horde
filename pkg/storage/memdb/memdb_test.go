package memdb

import (
	"os"
	"testing"

	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/eesrc/horde/pkg/storage/storetest"
	"github.com/stretchr/testify/require"
)

func TestMemDB(t *testing.T) {
	assert := require.New(t)

	defer os.Remove("persist.db")
	sourcedb, err := sqlstore.NewSQLMutexStore("sqlite3", ":memory:?_foreign_keys=1&_cache=shared", true, 1, 1)
	assert.NoError(err)

	destdb, err := sqlstore.NewSQLStore("sqlite3", "file::memory:?_cache=shared", true, 99, 99)
	assert.NoError(err)

	s := newCacheDB(sourcedb, destdb)
	storetest.StorageTest(t, s)
	storetest.SequenceTest(t, s)
}

func TestAllocMemDB(t *testing.T) {
	assert := require.New(t)

	defer os.Remove("persistAPN.db")
	sourcedb, err := sqlstore.NewSQLAPNStore("sqlite3", ":memory:?_foreign_keys=1&_cache=shared", true)
	assert.NoError(err)

	destdb, err := sqlstore.NewSQLAPNStore("sqlite3", "file::memory:?_cache=shared", true)
	assert.NoError(err)

	s := newAPNCacheDB(sourcedb, destdb)

	storetest.TestAPNStore(s, t)
}
