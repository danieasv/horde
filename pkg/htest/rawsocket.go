package htest
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
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"runtime"
	"strconv"

	"golang.org/x/sys/unix"
)

type iphdr struct {
	vhl   uint8
	tos   uint8
	iplen uint16
	id    uint16
	off   uint16
	ttl   uint8
	proto uint8
	csum  uint16
	src   [4]byte
	dst   [4]byte
}

type udphdr struct {
	src  uint16
	dst  uint16
	ulen uint16
	csum uint16
}

// pseudo header used for checksum calculation
type pseudohdr struct {
	ipsrc   [4]byte
	ipdst   [4]byte
	zero    uint8
	ipproto uint8
	plen    uint16
}

func checksum(buf []byte) uint16 {
	sum := uint32(0)

	for ; len(buf) >= 2; buf = buf[2:] {
		sum += uint32(buf[0])<<8 | uint32(buf[1])
	}
	if len(buf) > 0 {
		sum += uint32(buf[0]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	csum := ^uint16(sum)
	/*
	 * From RFC 768:
	 * If the computed checksum is zero, it is transmitted as all ones (the
	 * equivalent in one's complement arithmetic). An all zero transmitted
	 * checksum value means that the transmitter generated no checksum (for
	 * debugging or for higher level protocols that don't care).
	 */
	if csum == 0 {
		csum = 0xffff
	}
	return csum
}

func (h *iphdr) checksum() {
	h.csum = 0
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, h)
	h.csum = checksum(b.Bytes())
}

func (u *udphdr) checksum(ip *iphdr, payload []byte) {
	u.csum = 0
	phdr := pseudohdr{
		ipsrc:   ip.src,
		ipdst:   ip.dst,
		zero:    0,
		ipproto: ip.proto,
		plen:    u.ulen,
	}
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, &phdr)
	binary.Write(&b, binary.BigEndian, u)
	binary.Write(&b, binary.BigEndian, &payload)
	u.csum = checksum(b.Bytes())
}

// SendUDPWithSource (semi-) spoofs an UDP packet with a new source. The MAC address
// is unchanged. You must be root to call this
func SendUDPWithSource(destination string, source string, payload []byte) error {
	ipdststr, dstport, err := net.SplitHostPort(destination)
	if err != nil {
		return err
	}
	ipsrcstr, srcport, err := net.SplitHostPort(source)
	if err != nil {
		return err
	}
	sp, err := strconv.ParseUint(srcport, 10, 16)
	if err != nil {
		return err
	}
	dp, err := strconv.ParseUint(dstport, 10, 16)
	if err != nil {
		return err
	}
	udpsrc := uint(sp)
	udpdst := uint(dp)

	ipsrc := net.ParseIP(ipsrcstr)
	if ipsrc == nil {
		return fmt.Errorf("invalid source IP: %v", ipsrc)
	}
	ipdst := net.ParseIP(ipdststr)
	if ipdst == nil {
		return fmt.Errorf("invalid destination IP: %v", ipdst)
	}

	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_RAW, unix.IPPROTO_RAW)

	if err != nil || fd < 0 {
		return fmt.Errorf("error creating a raw socket: %v", err)
	}

	err = unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_HDRINCL, 1)
	if err != nil {
		return fmt.Errorf("error enabling IP_HDRINCL: %v", err)
	}

	ip := iphdr{
		vhl:   0x45,
		tos:   0,
		id:    0x1234, // the kernel overwrites id if it is zero
		off:   0,
		ttl:   64,
		proto: unix.IPPROTO_UDP,
	}
	copy(ip.src[:], ipsrc.To4())
	copy(ip.dst[:], ipdst.To4())
	// iplen and csum set later

	udp := udphdr{
		src: uint16(udpsrc),
		dst: uint16(udpdst),
	}
	// ulen and csum set later

	// just use an empty IPv4 sockaddr for Sendto
	// the kernel will route the packet based on the IP header
	addr := unix.SockaddrInet4{}

	udplen := 8 + len(payload)
	totalLen := 20 + udplen
	if totalLen > 0xffff {
		return fmt.Errorf("message is too large to fit into a packet: %v > %v", totalLen, 0xffff)
	}

	// the kernel will overwrite the IP checksum, so this is included just for
	// completeness
	ip.iplen = uint16(totalLen)
	ip.checksum()

	// the kernel doesn't touch the UDP checksum, so we can either set it
	// correctly or leave it zero to indicate that we didn't use a checksum
	udp.ulen = uint16(udplen)
	udp.checksum(&ip, payload)

	var b bytes.Buffer
	err = binary.Write(&b, binary.BigEndian, &ip)
	if err != nil {
		return fmt.Errorf("error encoding the IP header: %v", err)
	}
	err = binary.Write(&b, binary.BigEndian, &udp)
	if err != nil {
		return fmt.Errorf("error encoding the UDP header: %v", err)
	}
	err = binary.Write(&b, binary.BigEndian, &payload)
	if err != nil {
		return fmt.Errorf("error encoding the payload: %v", err)
	}
	bb := b.Bytes()

	/*
	 * For some reason, the IP header's length field needs to be in host byte order
	 * in OS X.
	 */
	if runtime.GOOS == "darwin" {
		bb[2], bb[3] = bb[3], bb[2]
	}
	err = unix.Sendto(fd, bb, 0, &addr)
	if err != nil {
		return fmt.Errorf("error sending the packet: %v", err)
	}

	err = unix.Close(fd)
	if err != nil {
		return fmt.Errorf("error closing the socket: %v", err)
	}
	return nil
}
