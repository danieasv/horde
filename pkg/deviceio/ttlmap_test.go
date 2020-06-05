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
	"testing"
	"time"

	"github.com/go-ocf/go-coap"
	"github.com/stretchr/testify/require"
)

func TestTTLMap(t *testing.T) {
	assert := require.New(t)

	m := newTTLMap(time.Millisecond * 10)

	assert.NotNil(m)

	ep := []string{"127.0.0.1:5683",
		"127.0.0.1:5684",
		"127.0.0.1:5685",
		"127.0.0.1:5686",
		"127.0.0.1:5687"}
	for _, v := range ep {
		c, err := coap.Dial("udp", v)
		assert.NoError(err)
		m.AddClientConnection(v, c)
	}
	cc := m.GetConnection(ep[0])
	assert.NotNil(cc)
	cc = m.GetConnection(ep[2])
	assert.NotNil(cc)
	cc = m.GetConnection("127.0.0.1:4711")
	assert.Nil(cc)
	time.Sleep(100 * time.Millisecond)

	cc = m.GetConnection(ep[0])
	assert.Nil(cc)

}
