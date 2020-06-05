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
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var coapPullCount *int32
var coapPushCount *int32
var udpCount *int32

func init() {
	coapPullCount = new(int32)
	coapPushCount = new(int32)
	udpCount = new(int32)

	atomic.StoreInt32(coapPullCount, 0)
	atomic.StoreInt32(coapPushCount, 0)
	atomic.StoreInt32(udpCount, 0)
}

func incrementPush() {
	atomic.AddInt32(coapPushCount, 1)
	sendAckPayload()
}

func incrementPull() {
	atomic.AddInt32(coapPullCount, 1)
	sendAckPayload()
}

func incrementUDP() {
	atomic.AddInt32(udpCount, 1)
	sendAckPayload()
}

var ackTarget string

func launchHTTPServer(cfg parameters) {
	ackTarget = cfg.UpstreamUDP

	http.HandleFunc("/coap-pull", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("%d\n", atomic.LoadInt32(coapPullCount))))
	})
	http.HandleFunc("/coap-push", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("%d\n", atomic.LoadInt32(coapPushCount))))
	})
	http.HandleFunc("/udp", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("%d\n", atomic.LoadInt32(udpCount))))
	})
	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Write([]byte("Stopping\n"))
			go func() {
				time.Sleep(1 * time.Second)
				os.Exit(0)
			}()
			return
		}
		w.Write([]byte("This is the stop resource\n"))
	})
	fmt.Println("HTTP runs on ", cfg.HTTP)
	go func() {
		if err := http.ListenAndServe(cfg.HTTP, nil); err != nil {
			fmt.Println("Unable to listen and serve on ", cfg.HTTP, " ", err)
			os.Exit(1)
		}
	}()
}

// Send "ACK" to the upstream source
func sendAckPayload() {
	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		fmt.Printf("Could not resolve local UDP address: %v\n", err)
		os.Exit(4)
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", ackTarget)
	if err != nil {
		fmt.Printf("Could not resolve remote UDP address: %v\n", err)
		os.Exit(4)
	}
	output, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		fmt.Printf("Couldn't dial UDP: %v\n", err)
		os.Exit(4)
	}
	defer output.Close()
	_, err = output.Write([]byte("ACK"))
	if err != nil {
		fmt.Printf("Couldn't dial UDP: %v\n", err)
		os.Exit(4)
	}
}
