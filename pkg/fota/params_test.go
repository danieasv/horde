package fota

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

	"github.com/ExploratoryEngineering/params"
	"github.com/stretchr/testify/require"
)

func TestParameters(t *testing.T) {
	assert := require.New(t)

	p := Parameters{FirmwareEndpoint: "coap://172.16.15.14:5683/fw"}

	host, port, path, err := p.GetFirmwareHostPortPath()
	assert.NoError(err)

	assert.Equal(host, "172.16.15.14")
	assert.Equal(port, 5683)
	assert.Equal(path, "fw")

	p.FirmwareEndpoint = "coap://some.host/otherpath"

	host, port, path, err = p.GetFirmwareHostPortPath()
	assert.NoError(err)
	assert.Equal(host, "some.host")
	assert.Equal(port, 5683)
	assert.Equal(path, "otherpath")

	p.FirmwareEndpoint = "coap://otherhost"

	host, port, path, err = p.GetFirmwareHostPortPath()
	assert.NoError(err)
	assert.Equal(host, "otherhost")
	assert.Equal(port, 5683)
	assert.Equal(path, "")

}

func TestDefaultParameters(t *testing.T) {
	assert := require.New(t)
	p := Parameters{}
	assert.NoError(params.NewFlag(&p, []string{}))
	assert.Equal("coap://172.16.15.14:5683/fw", p.FirmwareEndpoint)
	host, port, path, err := p.GetFirmwareHostPortPath()
	assert.NoError(err)
	assert.Equal("172.16.15.14", host)
	assert.Equal(5683, port)
	assert.Equal("fw", path)
	assert.Equal(30*time.Second, p.LWM2MPollInterval)
}
