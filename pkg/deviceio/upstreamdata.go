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

	"github.com/eesrc/horde/pkg/deviceio/rxtx"
)

// upstreamData is the upstream data from the devices. All UDP packets are
// processed and forwarded. They might be rejected by the horde service later.
type upstreamData struct {
	Msg  rxtx.Message
	Conn *net.UDPConn
}
