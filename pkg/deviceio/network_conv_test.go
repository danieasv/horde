package deviceio

//
// Copyright 2020 Telenor Digital AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNetToUintConversion(t *testing.T) {
	assert := require.New(t)

	_, net, err := net.ParseCIDR("10.1.0.0/16")
	assert.NoError(err)

	assert.Equal(*net, Uint64ToNetwork(NetworkToUint64(net)))

	num := uint64(0x0102030405060708)
	tmpnet := Uint64ToNetwork(num)
	assert.Equal(num, NetworkToUint64(&tmpnet))
}
