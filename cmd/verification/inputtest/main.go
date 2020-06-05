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
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ExploratoryEngineering/params"

	"github.com/dustin/go-coap"
	"github.com/telenordigital/nbiot-go"
)

type parameters struct {
	Endpoint     string        `param:"desc=HTTP API endpoint;default=http://127.0.0.1:8080"`
	Token        string        `param:"desc=API token;default="`
	CollectionID string        `param:"desc=Collection ID to use;default="`
	Repeat       int           `param:"desc=Number of messages to send;default=10"`
	Delay        time.Duration `param:"desc=Delay between messages;default=100ms"`
	UDPEndpoint  string        `param:"desc=UDP receiver endpoint;default=127.0.0.1:31415"`
	COAPEndpoint string        `param:"desc=CoAP endpoint;default=127.0.0.1:5683"`
}

func main() {
	var param parameters
	if err := params.NewEnvFlag(&param, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	c, err := nbiot.NewWithAddr(param.Endpoint, param.Token)
	if err != nil {
		panic(fmt.Sprintf("Unable to create API client: %v", err))
	}
	stream, err := c.CollectionOutputStream(param.CollectionID)
	if err != nil {
		panic(fmt.Sprintf("Unable to open output stream: %v", err))
	}

	go func() {
		fmt.Println("Waiting for messages...")
		defer stream.Close()
		received := 0
		for {
			msg, err := stream.Recv()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error receiving: %v", err)
				return
			}
			src := ""
			if msg.Payload[0] == 0 {
				src = "UDP"
			} else {
				src = "CoAP"
			}
			seq := binary.BigEndian.Uint16(msg.Payload[1:])
			timestamp := binary.BigEndian.Uint64(msg.Payload[3:])
			t := time.Unix(0, int64(timestamp))
			rtt := float64(time.Since(t)) / float64(time.Millisecond)

			fmt.Printf("%03d: %d bytes from %s (via %s). Seq = %d RTT = %6.3f ms\n", received, len(msg.Payload), msg.Device.ID, src, seq, rtt)
			received++
			if received == (2 * param.Repeat) {
				fmt.Printf("All messages received!\n")
				os.Exit(0)
				return
			}
		}
	}()

	sequence := uint16(0)
	for i := 0; i < param.Repeat; i++ {
		wait := time.After(param.Delay)

		buf := make([]byte, 11)
		buf[0] = 0
		binary.BigEndian.PutUint16(buf[1:], sequence)
		sequence++
		binary.BigEndian.PutUint64(buf[3:], uint64(time.Now().UnixNano()))
		sendUDP(param.UDPEndpoint, buf)

		buf[0] = 1
		binary.BigEndian.PutUint16(buf[1:], sequence)
		sequence++
		binary.BigEndian.PutUint64(buf[3:], uint64(time.Now().UnixNano()))
		sendCoAP(param.COAPEndpoint, sequence, buf)

		<-wait
	}
	// Wait for everuthing to drain
	time.Sleep(5 * time.Second)

	fmt.Println("Timed out waiting for messages")
	os.Exit(1)
}

func sendUDP(ep string, payload []byte) {
	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		panic(fmt.Sprintf("Could not resolve local UDP address: %v", err))
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", ep)
	if err != nil {
		panic(fmt.Sprintf("Could not resolve remote UDP address: %v", err))
	}
	output, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		panic(fmt.Sprintf("Couldn't dial UDP: %v", err))
	}
	defer output.Close()
	bytes, err := output.Write(payload)
	if err != nil {
		panic(fmt.Sprintf("Couldn't dial UDP: %v", err))
	}
	if bytes != len(payload) {
		panic(fmt.Sprintf("Sent %d bytes but %d were sent", bytes, len(payload)))
	}
}

func sendCoAP(ep string, msgid uint16, payload []byte) {
	req := coap.Message{
		Type:      coap.Confirmable,
		Code:      coap.POST,
		MessageID: msgid,
		Payload:   payload,
	}

	req.SetPathString("/input/test")

	c, err := coap.Dial("udp", ep)
	if err != nil {
		panic(fmt.Sprintf("Error dialing: %v", err))
	}

	rv, err := c.Send(req)
	if err != nil {
		panic(fmt.Sprintf("Error sending request: %v", err))
	}

	if rv.Type != coap.Acknowledgement {
		fmt.Fprintf(os.Stderr, "Got response: %+v\n", rv)
	}
}
