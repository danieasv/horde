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

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestTLVEncodeDecode(t *testing.T) {
	assert := require.New(t)

	idx := 0
	buf := make([]byte, 128)
	testString1 := "This is the first string"
	testString2 := "This is the second string"
	assert.NoError(encodeTLVString(1, &idx, buf, testString1))
	assert.NoError(encodeTLVString(2, &idx, buf, testString2))

	idx = 0
	outString1 := ""
	outString2 := ""
	assert.Equal(byte(1), buf[idx])
	idx++
	assert.NoError(decodeTLVString(&idx, buf, &outString1))
	assert.Equal(testString1, outString1)
	assert.Equal(byte(2), buf[idx])
	idx++
	assert.NoError(decodeTLVString(&idx, buf, &outString2))
	assert.Equal(outString2, testString2)
}

func TestReportEncoding(t *testing.T) {
	assert := require.New(t)

	original := Report{
		FirmwareVersion:  "1.2.3",
		ManufacturerName: "Man U Fac Tur Er Er Er",
		SerialNumber:     "double-oh-seven",
		ModelNumber:      "Das Model",
	}

	phd := Report{}

	buf := make([]byte, 1024)
	idx := 0
	assert.NoError(encodeTLVString(firmwareVersionID, &idx, buf, original.FirmwareVersion))
	assert.NoError(encodeTLVString(modelNumberID, &idx, buf, original.ModelNumber))
	assert.NoError(encodeTLVString(manufacturerNameID, &idx, buf, original.ManufacturerName))
	assert.NoError(encodeTLVString(serialNumberID, &idx, buf, original.SerialNumber))

	assert.NoError(phd.UnmarshalBinary(buf[:idx]))

	assert.Equal(original, phd)
}

func TestFwEndpoint(t *testing.T) {
	assert := require.New(t)

	fe := SimpleFOTAResponse{
		Host: "172.16.15.14",
		Port: 5683,
		Path: "d/e/v/i/c/e",
	}
	buf, err := fe.MarshalBinary()
	assert.NoError(err)
	spew.Dump(buf)
	assert.Equal(byte(hostID), buf[0])
	idx := 1
	outStr := ""
	assert.NoError(decodeTLVString(&idx, buf, &outStr))
	assert.Equal(fe.Host, outStr)
	assert.Equal(byte(portID), buf[idx])
	idx++
	assert.Equal(byte(4), buf[idx])
	idx += 5
	assert.Equal(byte(pathID), buf[idx])
	idx++
	assert.NoError(decodeTLVString(&idx, buf, &outStr))
	assert.Equal(fe.Path, outStr)

}
