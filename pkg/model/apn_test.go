package model

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
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAPN(t *testing.T) {
	NewAPN(1)
}

func TestNASRanges(t *testing.T) {
	assert := require.New(t)

	r := NewNASRanges(1, "apn1", 1, "nas1", "10.0.0.0/24")
	assert.Len(r.Ranges, 1)
	assert.Equal(r.APN.ID, 1)
	assert.Equal(r.Ranges[0].ID, 1)

	assert.Equal(r.GetNasID("nas1"), 1)
	assert.Equal(r.GetNasID("nas2"), -1)

	nas, found := r.ByIP(net.ParseIP("10.0.0.1"))
	assert.True(found)
	assert.Equal(1, nas.ID)
	nas, found = r.ByIP(net.ParseIP("127.0.0.1"))
	assert.False(found)
}

func TestFindNAS(t *testing.T) {
	assert := require.New(t)

	r := NewNASRanges(1, "apn1", 1, "nas1", "10.0.0.0/24")
	nas, found := r.Find(1)
	assert.True(found)
	assert.Equal(1, nas.ID)

	nas, found = r.Find(2)
	assert.False(found)
}
func TestInvalidNASRange(t *testing.T) {
	assert := require.New(t)

	r := NewNASRanges(1, "apn1", 1, "nas1", "10.0.0.0/24")
	r.Ranges = append(r.Ranges, NAS{ID: 2, Identifier: "nas2", CIDR: "abcd"})
	nas, found := r.ByIP(net.ParseIP("10.0.0.1"))
	assert.True(found)
	assert.Equal(1, nas.ID)

	nas, found = r.ByIP(net.ParseIP("172.16.0.100"))
	assert.False(found)
}
