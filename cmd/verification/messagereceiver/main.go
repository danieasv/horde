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
	"os"

	"github.com/ExploratoryEngineering/params"
	"github.com/eesrc/horde/pkg/utils"
)

type parameters struct {
	UDP          string `param:"desc=UDP listener interface;default=127.0.0.1:4711"`
	COAP         string `param:"desc=CoAP server endpoint;default=127.0.0.1:4712"`
	UpstreamUDP  string `param:"desc=Upstream UDP interface;default=127.0.0.1:31415"`
	UpstreamCOAP string `param:"desc=Upstream CoAP server;default=127.0.0.1:5683"`
	HTTP         string `param:"desc=Listen address for http interface;default=127.0.0.1:8282"`
}

func main() {
	var cfg parameters
	if err := params.NewEnvFlag(&cfg, os.Args[1:]); err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
	fmt.Println("messagereceiver")

	launchHTTPServer(cfg)

	// Launch the inputs. They will put the inputs on the channel, then
	// decrement the wait group when they're done and then exit.
	go listenOnUDP(cfg)
	go launchCoAPServer(cfg)
	go readCoAPPull(cfg)

	// Wait for the file to write
	fmt.Println("Wait for signal")
	utils.WaitForSignal()
}
