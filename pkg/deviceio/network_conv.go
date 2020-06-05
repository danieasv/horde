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

	"github.com/eesrc/horde/pkg/utils"
)

// net.IPNet conversion to and from uint64.

// NetworkToUint64 converts the net.IPNet structure into a single 64-bit integer.
// The high 32 bits contains the IPv4 address and the lower 32 bits contains the
// mask bits.
func NetworkToUint64(net *net.IPNet) uint64 {
	ret := uint64(utils.AtonIPv4(net.IP.To4())) << 32

	ret |= uint64(net.Mask[0]) << 24
	ret |= uint64(net.Mask[1]) << 16
	ret |= uint64(net.Mask[2]) << 8
	ret |= uint64(net.Mask[3])

	return ret
}

// Uint64ToNetwork converts an uint64 value with the IPv4 and mask bits into a
// net.IPNet structure
func Uint64ToNetwork(val uint64) net.IPNet {
	ip := uint32(val >> 32)
	mask := net.IPv4Mask(byte((val>>24)&0xFF),
		byte((val>>16)&0xFF),
		byte((val>>8)&0xFF),
		byte(val&0xFF))
	return net.IPNet{IP: utils.NtoaIPv4(ip), Mask: mask}
}
