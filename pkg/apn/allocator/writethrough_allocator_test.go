package allocator

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
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/stretchr/testify/require"
)

func TestWritethroughAllocator(t *testing.T) {
	assert := require.New(t)

	store := sqlstore.NewMemoryAPNStore()
	assert.NoError(store.CreateAPN(model.APN{ID: 1, Name: "test.apn"}))
	assert.NoError(store.CreateNAS(model.NAS{
		ID:         1,
		ApnID:      1,
		CIDR:       "10.2.1.1/24",
		Identifier: "NAS1",
	}))

	apnCache, err := storage.NewAPNCache(store)
	assert.NoError(err)
	a, err := NewWriteThroughAllocator(apnCache, store)
	assert.NoError(err)

	testAllocator(t, 1, a)
}
