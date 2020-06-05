package memdb

import (
	"testing"

	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/stretchr/testify/require"
)

func TestPriming(t *testing.T) {
	assert := require.New(t)

	src, err := sqlstore.NewSQLStore("sqlite3", "file::memory:?shared_cache=true", true, 1, 1)
	assert.NoError(err)
	assert.NotNil(src)

	memDB, err := PrimeMemoryCache(src)
	assert.NoError(err)
	assert.NotNil(memDB)

	srcAPN, err := sqlstore.NewSQLAPNStore("sqlite3", "file::memory:?shared_cache=true", true)
	assert.NoError(err)
	assert.NotNil(srcAPN)

	memAPN, err := PrimeAPNCache(srcAPN)
	assert.NoError(err)
	assert.NotNil(memAPN)

}
