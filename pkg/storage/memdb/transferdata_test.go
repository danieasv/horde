package memdb

import (
	"database/sql"
	"testing"

	"github.com/eesrc/horde/pkg/htest"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/stretchr/testify/require"
)

func TestTransfer(t *testing.T) {
	const userCount = 10
	const devicesPerCollection = 10
	assert := require.New(t)

	sourcedb, err := sqlstore.NewSQLStore("sqlite3", "file::memory:?cache=shared", true, 1, 1)
	assert.NoError(err)

	sourceconn := sqlstore.SQLConnection(sourcedb)
	assert.NotNil(sourceconn)

	destconn, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(err)
	schema := sqlstore.NewSchema("sqlite3", sqlstore.DBSchema)

	assert.NoError(schema.Create(destconn))

	settings := htest.DefaultGeneratorSettings
	settings.DevicesPerCollection = devicesPerCollection
	assert.NoError(htest.Generate(userCount, settings, sourcedb))

	assert.NoError(MoveData("hordeuser", sourceconn, destconn, ""))
	assert.NoError(MoveData("token", sourceconn, destconn, ""))
	assert.NoError(MoveData("team", sourceconn, destconn, ""))
	assert.NoError(MoveData("member", sourceconn, destconn, ""))
	assert.NoError(MoveData("collection", sourceconn, destconn, ""))
	assert.NoError(MoveData("output", sourceconn, destconn, ""))
	assert.NoError(MoveData("device", sourceconn, destconn, ""))

	destdb, err := sqlstore.NewSQLStoreWithConnection(destconn, 2, 2)
	assert.NoError(err)

	// Pull the outputs from the source store and verify that the corresponding
	// collections exist in the target store
	outputs, err := destdb.OutputListAll()
	assert.NoError(err)
	assert.Equal(htest.OutputCount(userCount, settings), len(outputs), "Number of outputs should be the same")
}
