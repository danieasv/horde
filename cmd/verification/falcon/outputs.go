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
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	nbiot "github.com/telenordigital/nbiot-go"
)

type outputs struct {
	Messages []nbiot.OutputDataMessage `json:"messages"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Unable to read body: %v", err)
		return
	}
	defer r.Body.Close()
	values := outputs{}
	if err := json.Unmarshal(body, &values); err != nil {
		log.Printf("Unable to unmarshal payload: %v (body: %s) (%+v)", err, string(body), r)
	}

	for _, v := range values.Messages {
		seq := binary.BigEndian.Uint32(v.Payload)
		t := binary.BigEndian.Uint64(v.Payload[4:])
		rtt := time.Now().UnixNano() - int64(t)
		log.Printf("RTT time for message from device %s with seq %d is %d us", v.Device.ID, seq, rtt/int64(time.Microsecond))
	}
	w.Write([]byte("I just got the messages back from you and I'm writing something you don't care about."))
}

func startWebserver(ep string) {
	handler := http.NewServeMux()
	handler.HandleFunc("/webhook", webhookHandler)
	go func() {
		if err := http.ListenAndServe(ep, handler); err != nil {
			log.Printf("Got error launching local web server: %v", err)
		}
	}()
}

func startUDPServer() {
	pc, err := net.ListenPacket("udp", ":4711")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	go func() {
		defer pc.Close()
		buf := make([]byte, 1024)
		for {
			n, _, err := pc.ReadFrom(buf)
			if err != nil {
				continue
			}
			log.Printf("Got %d bytes from output: %+v.", n, buf[0:n])
		}
	}()
}
