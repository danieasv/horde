package main

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
	"fmt"
	"log"
	"net"

	"github.com/dustin/go-coap"
)

func launchCoAPServer(cfg parameters) {
	handler := func(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
		fmt.Printf("CoAP push: Got %d bytes\n", len(m.Payload))
		var res *coap.Message
		if m.IsConfirmable() {
			res = &coap.Message{
				Type:      coap.Acknowledgement,
				Code:      coap.Valid,
				MessageID: m.MessageID,
				Token:     m.Token,
			}
		}
		incrementPush()
		return res
	}
	mux := coap.NewServeMux()
	mux.Handle("/push", coap.FuncHandler(handler))
	fmt.Println("CoAP push listening on ", cfg.COAP)
	go func() {
		log.Fatal(coap.ListenAndServe("udp", cfg.COAP, mux))
	}()
}
