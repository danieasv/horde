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
	"sync"
	"time"

	"github.com/go-ocf/go-coap"
)

const defaultClientTimeout = time.Minute * 90

// The LwM2M clients are a particular breed of clients. They are behaving like
// CoAP servers but in reality they're just clients that talk through a
// UDP socket. Any communication that isn't originating from the server will
// be discarded. This poses a few problems for us when we're sending requests
// back to the clients since we have to keep a map of the connections we've
// been using to serve requests. It's PITA but we have to handle it for the
// foreseeable future. This means that the lwm2m exchanges are quite brittle
// wrt restarts (I expect nothing less when telco standards and embedded
// software meet).
//
// ttlMap keeps a map of UDP client connections for N minutes. If an exchange
// is initiated to the client the client connection is reused. After N minutes
// the connections are released. All client connections are cached.
type ttlMap struct {
	clients map[string]ttlEntry
	mutex   *sync.Mutex
	expire  time.Duration
}

type ttlEntry struct {
	Access time.Time
	Client *coap.ClientConn
}

func (t *ttlMap) AddClientConnection(ep string, cc *coap.ClientConn) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.clients[ep] = ttlEntry{Access: time.Now(), Client: cc}
}

func (t *ttlMap) GetConnection(ep string) *coap.ClientConn {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	res, ok := t.clients[ep]
	if !ok {
		return nil
	}
	res.Access = time.Now()
	t.clients[ep] = res
	return res.Client
}

func (t *ttlMap) start() {
	for {
		t.mutex.Lock()
		for k, v := range t.clients {
			if time.Since(v.Access) > t.expire {
				delete(t.clients, k)
			}
		}
		t.mutex.Unlock()
		time.Sleep(2 * t.expire)
	}
}

// newTTLMap creates a new TTL map.
func newTTLMap(expireTime time.Duration) *ttlMap {
	ret := &ttlMap{
		clients: make(map[string]ttlEntry),
		mutex:   &sync.Mutex{},
		expire:  expireTime,
	}
	go ret.start()
	return ret
}
