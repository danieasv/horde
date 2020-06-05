package htest

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
	"encoding/binary"
	"fmt"

	"github.com/eesrc/horde/pkg/fota"
)

// Encoding and decoding functions for the device side of the simple FOTA
// process. Common sense would dictate that "this belongs with the encoding"
// but to avoid any integration issues this lives separately from the server
// side encoding and decoding. If there is any changes in the interface it
// will trigger an error elsewhere in the code base and hopefully that would
// raise some red flags.
//
// I hope.

// Identifiers for the TLV buffers in the report from the device. The
// identifiers are held in the first byte then the length, then the payload.
const (
	FirmwareID     = 1
	ManufacturerID = 2
	ModelID        = 4
	SerialID       = 3
)

// Identifiers for the TLV buffer sent back to the device.
const (
	hostID      = 1
	portID      = 2
	pathID      = 3
	availableID = 4
)

// TLVBuffer is a type to decode TLV payloads
type TLVBuffer struct {
	idx    int
	buffer []byte
}

// NewTLVBuffer creates a new TLV buffer
func NewTLVBuffer(length int) *TLVBuffer {
	return &TLVBuffer{
		idx:    0,
		buffer: make([]byte, length),
	}
}

// Begin starts encoding
func (t *TLVBuffer) Begin() {
	t.idx = 0
	for i := range t.buffer {
		t.buffer[i] = 0
	}
}

// EncodeTLVString encodes a string into a TLV buffer
func (t *TLVBuffer) EncodeTLVString(id byte, value string) {
	t.buffer[t.idx] = id
	t.idx++
	t.buffer[t.idx] = byte(len(value))
	t.idx++
	for _, ch := range value {
		t.buffer[t.idx] = byte(ch)
		t.idx++
	}
}

// Buffer returns the encoded buffer
func (t *TLVBuffer) Buffer() []byte {
	return t.buffer
}

func decodeTLVString(buf []byte, idx *int) string {
	len := buf[*idx]
	*idx++
	ret := ""
	for i := byte(0); i < len; i++ {
		ret = ret + string(buf[*idx])
		*idx++
	}
	return ret
}

func decodeTLVUint32(buf []byte, idx *int) uint32 {
	ret := uint32(0)
	if buf[*idx] != 4 {
		panic("int32 field length is not 4")
	}
	*idx++
	ret = binary.BigEndian.Uint32(buf[*idx:])
	*idx += 4
	return ret
}

func decodeTLVBool(buf []byte, idx *int) bool {
	ret := false
	if buf[*idx] != 1 {
		panic("bool field lenght is not 1")
	}
	*idx++
	if buf[*idx] == 1 {
		ret = true
	}
	*idx++
	return ret
}

// DecodeSimpleFOTAResponse decodes a simple FOTA response
func DecodeSimpleFOTAResponse(buf []byte) (fota.SimpleFOTAResponse, error) {
	ret := fota.SimpleFOTAResponse{}

	idx := 0

	for idx < len(buf) {
		switch buf[idx] {
		case hostID:
			idx++
			ret.Host = decodeTLVString(buf, &idx)
		case portID:
			idx++
			ret.Port = decodeTLVUint32(buf, &idx)
		case pathID:
			idx++
			ret.Path = decodeTLVString(buf, &idx)
		case availableID:
			idx++
			ret.ImageAvailable = decodeTLVBool(buf, &idx)
		default:
			return ret, fmt.Errorf("unknown id %d at pos %d", buf[idx], idx)
		}
	}
	return ret, nil
}
