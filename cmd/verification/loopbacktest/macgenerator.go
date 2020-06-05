package main
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
	"net"
)

// macGenerator is a MAC address generator based on a prefix. The MAC
// address is 6 bytes (aka 48 bits)
type macGenerator struct {
	prefix  [3]byte
	counter uint32
}

func newMacGenerator(prefix [3]byte) macGenerator {
	return macGenerator{prefix, 0}
}

func (m *macGenerator) NextMAC() net.HardwareAddr {
	m.counter++
	ret := make([]byte, 6)
	binary.BigEndian.PutUint32(ret[2:], m.counter)
	ret[0] = m.prefix[0]
	ret[1] = m.prefix[1]
	ret[2] = m.prefix[2]

	return net.HardwareAddr(ret)
}
