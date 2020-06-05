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
	"testing"
)

// This data is pulled from the spec (http://www.openmobilealliance.org/release/LightweightM2M/V1_1-20180710-A/OMA-TS-LightweightM2M_Core-V1_1-20180710-A.pdf)
// with a bugfix in the 2nd field (yes it *does* contain a spelling error in the spec. I'm impressed.).
var buffer = []byte{
	0x08, 0x00, 0x79,
	0xC8, 0x00, 0x14, 0x4F, 0x70, 0x65, 0x6E, 0x20, 0x4D, 0x6F, 0x62, 0x69, 0x6C, 0x65, 0x20, 0x41, 0x6C, 0x6C, 0x69, 0x61, 0x6E, 0x63, 0x65,
	0xC8, 0x01, 0x16, 0x4C, 0x69, 0x67, 0x68, 0x74, 0x77, 0x65, 0x69, 0x67, 0x74, 0x20, 0x4D, 0x32, 0x4D, 0x20, 0x43, 0x6C, 0x69, 0x65, 0x6E, 0x74, 0x74,
	0xC8, 0x02, 0x09, 0x33, 0x34, 0x35, 0x30, 0x30, 0x30, 0x31, 0x32, 0x33,
	0xC3, 0x03, 0x31, 0x2E, 0x30,
	0x86, 0x06,
	0x41, 0x00, 0x01,
	0x41, 0x01, 0x05,
	0x88, 0x07, 0x08,
	0x42, 0x00, 0x0E, 0xD8,
	0x42, 0x01, 0x13, 0x88,
	0x87, 0x08,
	0x41, 0x00, 0x7D,
	0x42, 0x01, 0x03, 0x84,
	0xC1, 0x09, 0x64,
	0xC1, 0x0A, 0x0F,
	0x83, 0x0B,
	0x41, 0x00, 0x00,
	0xC4, 0x0D, 0x51, 0x82, 0x42, 0x8F,
	0xC6, 0x0E, 0x2B, 0x30, 0x32, 0x3A, 0x30, 0x30,
	0xC1, 0x10, 0x55,
}

func TestTLVBuffer(t *testing.T) {
	buf := NewTLVBuffer(buffer)

	if buf.GetPayload(0, 0).String() != "Open Mobile Alliance" {
		t.Fatal("Invalid 0 0")
	}

	// Notice spelling -- the example said "Lightweigt M2M Client" originally.
	if buf.GetPayload(0, 1).String() != "Lightweigt M2M Clientt" {
		t.Fatal("Invalid 0 1: ", buf.GetPayload(0, 1).String())
	}

	if buf.GetPayload(0, 6, 0).Byte() != 1 {
		t.Fatal("Invalid sub-sub")
	}

	if buf.GetPayload(999) != nil {
		t.Fatal("Expected nil response")
	}

	// Get list of payloads for 0, 7
	voltages := buf.GetPayloadList(0, 7)
	if len(voltages) != 2 {
		t.Fatal("Expected 2 but got ", voltages)
	}
	if voltages[0].Uint16() != 0x0ed8 || voltages[1].Uint16() != 0x1388 {
		t.Fatal("Invalids ", voltages)
	}
}

func TestTLVHeader(t *testing.T) {
	h := tlvHeader{}
	h.ID = 1
	h.Length = 3
	h.FieldType = 3
	buf, _ := h.MarshalBinary()
	if len(buf) != 2 {
		t.Fatal("Buffer is incorrect length: ", len(buf), buf)
	}
	// Should be 11000011 = 195
	if buf[0] != 195 {
		t.Fatalf("Not header I expected: %v, (%+v)", buf[0], buf)
	}
	// ID should be the 2nd byte
	if buf[1] != 1 {
		t.Fatalf("Incorrect ID: %v", buf[1])
	}

	h.ID = 0xFFEE
	h.Length = 6
	h.FieldType = 3
	buf, _ = h.MarshalBinary()
	if len(buf) != 3 {
		t.Fatal("Buffer is incorrect length: ", len(buf), buf)
	}
	if buf[1] != 0xFF || buf[2] != 0xEE {
		t.Fatalf("ID isn't set: %+v", buf)
	}
	h.Length = 257
	buf, _ = h.MarshalBinary()
	if len(buf) != 5 {
		t.Fatal("Buffer is incorrect length: ", len(buf), buf)
	}
	h.Length = 0xFFFFFE
	buf, _ = h.MarshalBinary()
	if len(buf) != 6 {
		t.Fatal("Buffer is incorrect lenght: ", len(buf), buf)
	}
}
func TestTLVEncode(t *testing.T) {
	const (
		s1 = "Hello"
		s2 = "Hello there I'm a really long string"
	)

	buf := EncodeString(5, s1)
	b1 := NewTLVBuffer(buf)
	p1 := b1.GetPayload(5)
	if p1.String() != s1 {
		t.Fatal()
	}
	if p1.ID[0] != 5 {
		t.Fatal()
	}

	buf = EncodeString(5, s2)
	b2 := NewTLVBuffer(buf)
	p2 := b2.GetPayload(5)
	if p2.String() != s2 {
		t.Fatal()
	}
}
