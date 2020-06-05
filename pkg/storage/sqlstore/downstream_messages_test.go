package sqlstore

import (
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/stretchr/testify/require"
)

// Test the downstream store. The messages themselves are protobuf messages
// in binary form so there's no need to test extensively for content.
func TestDownstreamStore(t *testing.T) {
	assert := require.New(t)

	params := Parameters{
		ConnectionString: ":memory:",
		Type:             "sqlite3",
		CreateSchema:     true,
	}

	store, err := NewDownstreamStore(NewMemoryStore(), params, 1, 1)
	assert.NoError(err)

	assert.NotNil(store)

	// Create a message
	assert.NoError(store.Create(1, 1, 1, 1, model.CoAPTransport, []byte("coap-1")))
	// Duplicate message ID should yield error
	assert.Error(store.Create(1, 1, 1, 1, model.CoAPTransport, []byte("coap-1")))

	// Create 3 more CoAP messages
	assert.NoError(store.Create(1, 1, 1, 2, model.CoAPTransport, []byte("coap-2")))
	assert.NoError(store.Create(1, 1, 1, 3, model.CoAPTransport, []byte("coap-3")))
	assert.NoError(store.Create(1, 1, 1, 4, model.CoAPTransport, []byte("coap-4")))

	// Create 3 UDP messages
	assert.NoError(store.Create(1, 1, 1, 5, model.UDPTransport, []byte("udp-5")))
	assert.NoError(store.Create(1, 1, 1, 6, model.UDPTransport, []byte("udp-6")))
	assert.NoError(store.Create(1, 1, 1, 7, model.UDPTransport, []byte("udp-7")))

	// Retrieve a CoAP message. It should be the first
	m, buf, err := store.Retrieve(1, 1, model.CoAPTransport)
	assert.NoError(err)
	assert.NotNil(buf)
	assert.Equal(model.MessageKey(1), m)

	// Retrieve for an APN that does not exist
	_, _, err = store.Retrieve(2, 1, model.CoAPTransport)
	assert.Equal(storage.ErrNotFound, err)

	// Remove the first message
	assert.NoError(store.Delete(1))
	assert.Equal(storage.ErrNotFound, store.Delete(1))

	// Retrieve UDP message. It should be the first added
	m, buf, err = store.Retrieve(1, 1, model.UDPTransport)
	assert.NoError(err)
	assert.NotNil(buf)
	assert.Equal(model.MessageKey(5), m)

	// Remove the first UDP message
	assert.NoError(store.Delete(5))

	// Retrieve for unknown device
	_, _, err = store.RetrieveByDevice(2, model.CoAPTransport)
	assert.Equal(storage.ErrNotFound, err)

	// ... known device
	m, _, err = store.RetrieveByDevice(1, model.CoAPTransport)
	assert.NoError(err)
	assert.Equal(model.MessageKey(2), m)

	// Get the next UDP message
	m, buf, err = store.Retrieve(1, 1, model.UDPTransport)
	assert.NoError(err)
	assert.NotNil(buf)
	assert.Equal("udp-6", string(buf))
	assert.Equal(model.MessageKey(6), m)

	// Remove it
	assert.NoError(store.Delete(m))

	// next should be #7
	m, _, err = store.Retrieve(1, 1, model.UDPTransport)
	assert.NoError(err)
	assert.Equal(model.MessageKey(7), m)

	assert.NoError(store.Delete(m))

	assert.NoError(store.Delete(4))
	assert.NoError(store.Delete(3))
}

func TestDownstreamStoreMessageQueuing(t *testing.T) {
	assert := require.New(t)

	// This will create a db that persists across instances
	params := Parameters{
		ConnectionString: "file::memory:?cache=shared",
		Type:             "sqlite3",
		CreateSchema:     true,
	}

	store, err := NewDownstreamStore(NewMemoryStore(), params, 1, 1)
	assert.NoError(err)
	assert.NotNil(store)

	const messageCount = model.MessageKey(10)
	// Line up 10x messages
	for id := model.MessageKey(0); id < messageCount; id++ {
		assert.NoError(store.Create(1, 1, 1, id, model.CoAPTransport, []byte("coapdata")))
	}

	// Create a new store. This should contain the same messages.
	store, err = NewDownstreamStore(NewMemoryStore(), params, 1, 1)
	assert.NoError(err)
	assert.NotNil(store)

	var retrieved []model.MessageKey
	// Pull the first five messages
	for i := 0; i < int(messageCount); i++ {
		key, _, err := store.RetrieveByDevice(1, model.CoAPTransport)
		assert.NoError(err)
		assert.NotContains(retrieved, key)
		retrieved = append(retrieved, key)
	}

	// Release every odd message, remove every even-numbered message
	for i := 0; i < int(messageCount); i++ {
		if i%2 == 0 {
			assert.NoError(store.Delete(model.MessageKey(i)))
		} else {
			store.Release(model.MessageKey(i))
		}
	}

	retrieved = make([]model.MessageKey, 0)
	// The store should now contain five messages
	for i := 0; i < int(messageCount/2); i++ {
		key, _, err := store.RetrieveByDevice(1, model.CoAPTransport)
		assert.NoError(err)
		assert.NotContains(retrieved, key)
		retrieved = append(retrieved, key)
	}
	_, _, err = store.RetrieveByDevice(1, model.CoAPTransport)
	assert.Equal(storage.ErrNotFound, err)
}
