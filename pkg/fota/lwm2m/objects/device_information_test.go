package objects

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
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeviceInformation(t *testing.T) {
	assert := require.New(t)

	b := NewTLVBuffer(buffer[3:])
	m := NewDeviceInformation(b)

	assert.Equal("1.0", m.FirmwareVersion)
	assert.Equal(uint8(0x64), m.BatteryLevel)
	assert.Equal("Open Mobile Alliance", m.Manufacturer)
	assert.Equal("Lightweigt M2M Clientt", m.ModelNumber)
	assert.Equal("345000123", m.SerialNumber)

	buf := m.Buffer()

	b2 := NewTLVBuffer(buf)
	m2 := NewDeviceInformation(b2)

	assert.EqualValues(m, m2)
}
