package metrics

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

func TestRADIUSCounters(t *testing.T) {
	assert := require.New(t)

	store := sqlstore.NewMemoryAPNStore()

	apn1 := model.NewAPN(1)
	apn1.Name = "mda.01"
	assert.NoError(store.CreateAPN(apn1))

	apn2 := model.NewAPN(2)
	apn2.Name = "mda.02"
	assert.NoError(store.CreateAPN(apn2))

	assert.NoError(store.CreateNAS(model.NAS{ID: 1, Identifier: "TNAS01", CIDR: "10.0.0.0/16", ApnID: 1}))
	assert.NoError(store.CreateNAS(model.NAS{ID: 2, Identifier: "TNAS02", CIDR: "10.1.0.0/16", ApnID: 1}))
	assert.NoError(store.CreateNAS(model.NAS{ID: 3, Identifier: "TNAS03", CIDR: "10.2.0.0/16", ApnID: 2}))
	assert.NoError(store.CreateNAS(model.NAS{ID: 4, Identifier: "TNAS04", CIDR: "10.3.0.0/16", ApnID: 2}))

	apnConfig, err := storage.NewAPNCache(store)
	assert.NoError(err)

	c := NewRADIUSCounters()
	c.Start(apnConfig)

	c.AcceptRequest("TNAS01")
	c.AcceptRequest("TNAS02")
	c.AcceptRequest("TNAS03")
	c.AcceptRequest("TNAS04")
	c.AcceptRequest("TNAS99")

	c.RejectRequest("TNAS01")
	c.IPAllocated(("TNAS02"))
	c.IPReleased("TNAS03")
	c.IPReused("TNAS03")
	c.UpdateIPAllocation("TNAS01", 0, 99)
}
