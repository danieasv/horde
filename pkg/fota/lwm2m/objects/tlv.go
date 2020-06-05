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
	"encoding/binary"
)

// TLVBuffer does (T)LV decoding of LwM2M response objects.
//
// The LwM2M spec says it's "TLV" but in reality the fields are just LV, ie just
// and ID, length and a byte buffer. The type is inferred by the ID and there's
// no type information embedded in the buffer. We only need to access the byte
// buffers to decode the data structs.
//
// The hierarchy is fairly simple; there's a top level Object ID.
// Each element inside of that object has a ID. Some of those elements might
// contain additional sub-elements. At most there's three elements to keep
// track of. The first top level object has ID 0. The type of the data structure
// is known up front. Each element inside of that object has an unique ID. Each
// element might consist of sub-elements which is numbered individually, ie we
// might have the top level object 0 with the elements 0, 1, 2, 3... and element
// 2 might contain additional elements numbered 0, 1, 3.
//
// Objects are managed in the OMA registry at
// http://www.openmobilealliance.org/wp/OMNA/LwM2M/LwM2MRegistry.html
//
// The standard objects are defined in
// http://www.openmobilealliance.org/release/LightweightM2M/V1_1-20180710-A/OMA-TS-LightweightM2M_Core-V1_1-20180710-A.pdf
//
// Typically we are just interested in the binary payload for a given ID or
// the payloads in a sequence of IDs.
type TLVBuffer struct {
	// ID is the object ID
	ID int
	// Resources hold all of the resources
	Resources []Payload
}

// GetPayload returns the payload with the specified ID(s). Nil is returned if
// the payload doesn't exist.
func (t *TLVBuffer) GetPayload(id ...int) *Payload {
	for _, v := range t.Resources {
		if v.matches(true, id) {
			return &v
		}
	}
	return nil
}

// GetPayloadList returns a list of payloads matching the IDs
func (t *TLVBuffer) GetPayloadList(id ...int) []Payload {
	var ret []Payload
	for _, v := range t.Resources {
		if v.matches(false, id) {
			ret = append(ret, v)
		}
	}
	return ret
}

// Payload is a single binary payload. The ID field contains a set of IDs that
// identify the payload.
type Payload struct {
	// ID is the array of IDs that points to this payload. The first ID will be
	// the enclosing object
	ID []int
	// Buffer is the payload itself
	Buffer []byte
}

func (p *Payload) matches(exactMatch bool, id []int) bool {
	if len(p.ID) < len(id) {
		return false
	}
	if exactMatch && len(id) != len(p.ID) {
		return false
	}
	for i := range id {
		if id[i] != p.ID[i] {
			return false
		}
	}
	return true
}

// String returns the payload as a string
func (p Payload) String() string {
	return string(p.Buffer)
}

// Byte returns the first byte of the payload
func (p *Payload) Byte() byte {
	return byte(p.Buffer[0])
}

// Uint16 returns the uint16 value of the payload
func (p *Payload) Uint16() uint16 {
	return binary.BigEndian.Uint16(p.Buffer)
}

// NewTLVBuffer decodes a byte buffer
func NewTLVBuffer(buffer []byte) TLVBuffer {
	if buffer == nil {
		return TLVBuffer{}
	}
	return TLVBuffer{Resources: decodeTLVBuffer([]int{}, buffer)}
}

func decodeTLVBuffer(id []int, buffer []byte) []Payload {
	index := 0
	ret := make([]Payload, 0)
	for index < len(buffer) {
		header, offs := newTLVHeader(buffer[index:])
		index += offs
		switch header.FieldType {
		case 0: // Object Instance, b00
			ret = append(ret, decodeTLVBuffer(append(id, header.ID), buffer[index:index+header.Length])...)
		case 1: // Resource Instance, b10
			ret = append(ret, Payload{ID: append(id, header.ID), Buffer: buffer[index : index+header.Length]})
		case 2: // Multiple Resources, b01
			ret = append(ret, decodeTLVBuffer(append(id, header.ID), buffer[index:index+header.Length])...)
		case 3: // Resource With Value, b11
			ret = append(ret, Payload{ID: append(id, header.ID), Buffer: buffer[index : index+header.Length]})
		}
		index += header.Length
	}
	return ret
}

// Header holds the header information for each type (ID, Length, type)
type tlvHeader struct {
	FieldType byte
	Length    int
	ID        int
}

// NewTLVHeader returns the header and the number of bytes read by
// the decoder. The header might be anything from 2 to 5 bytes.
// The header byte is split into four parts
// Bits 6-7: Type (object instance, multiple resource, resource ...)
// Bit 5: ID indicator: 0: 1 byte, 1: 2 bytes
// Bits 3-4: Lenght indicator (00 = in header, 01 = 1 byte, 10 = 2 bytes, 11 = 3 bytes)
//  Bits 0-2: Length of type (if it fits)
func newTLVHeader(buffer []byte) (tlvHeader, int) {
	ret := tlvHeader{}
	index := 1
	headerByte := buffer[0]
	ret.FieldType = headerByte >> 6     // 11000000
	idSize := (headerByte >> 5) & 1     // 00100000
	lengthSize := (headerByte >> 3) & 3 // 00011000
	ret.Length = int(headerByte & 7)    // 00000111

	switch idSize {
	case 0:
		ret.ID = int(buffer[index])
		index++
	case 1:
		ret.ID = int(binary.BigEndian.Uint16(buffer[index:]))
		index += 2
	}
	switch lengthSize {
	case 0:
		// Length is already set
	case 1:
		ret.Length = int(buffer[index])
		index++
	case 2:
		ret.Length = int(binary.BigEndian.Uint16(buffer[index:]))
		index += 2
	case 3:
		ret.Length = int(binary.BigEndian.Uint32(buffer[index:]))
		ret.Length = ret.Length >> 8
		index += 3
	}
	return ret, index
}

func (t *tlvHeader) MarshalBinary() ([]byte, error) {
	// worst case is 6 bytes (1 header, 2 id, 3 length) plus extra for the 32-bit length field
	buf := make([]byte, 7)

	lengthOffset := 2
	idLength := 0

	buf[1] = byte(t.ID & 0xFF)
	if t.ID > 0xFF {
		// Set ID flag and put 16 bit int into buffer
		lengthOffset = 3
		binary.BigEndian.PutUint16(buf[1:], uint16(t.ID))
	}

	headerLength := t.Length
	idLength = 0
	if t.Length > 0xFFFF {
		idLength = 3
		binary.BigEndian.PutUint32(buf[lengthOffset:], uint32(t.Length<<8))
		headerLength = 0
	} else if t.Length > 0xFF {
		idLength = 2
		binary.BigEndian.PutUint16(buf[lengthOffset:], uint16(t.Length))
		headerLength = 0
	} else if t.Length > 0x7 {
		idLength = 1
		buf[lengthOffset] = byte(t.Length)
		headerLength = 0
	}
	buf[0] = (t.FieldType << 6) + byte((idLength&3)<<3) + byte(headerLength&0x7)
	return buf[:lengthOffset+idLength], nil
}

// EncodeString encodes a string into a TLV byte buffer
func EncodeString(id int, value string) []byte {
	var header tlvHeader
	header.ID = id
	header.Length = len(value)
	header.FieldType = 3 // Resource with value
	buf, _ := header.MarshalBinary()
	return append(buf, []byte(value)...)
}

// EncodeBytes encodes a byte array into a TLV byte buffer
func EncodeBytes(id int, value []byte) []byte {
	var header tlvHeader
	header.ID = id
	header.Length = len(value)
	header.FieldType = 3
	buf, _ := header.MarshalBinary()
	return append(buf, value...)
}
