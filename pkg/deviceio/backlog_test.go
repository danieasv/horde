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
	"os"
	"testing"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/deviceio/rxtx"
	"github.com/stretchr/testify/require"
)

func makeRandomMessage() upstreamData {
	return upstreamData{
		Msg: rxtx.Message{
			Type:          rxtx.MessageType_UDP,
			Timestamp:     time.Now().UnixNano(),
			RemoteAddress: net.ParseIP("127.0.0.1"),
			RemotePort:    8086,
			LocalPort:     31415,
			Payload:       make([]byte, 10),
		},
	}
}

func TestBacklog(t *testing.T) {
	logging.SetLogLevel(logging.WarningLevel)
	const messageCount = 10

	assert := require.New(t)
	defer os.Remove(backlogUDPDatabase)

	backlog, err := newMessageBacklog(backlogUDPDatabase)
	assert.NotNil(backlog)
	assert.NoError(err)

	assert.NoError(backlog.Reset())

	for i := 0; i < messageCount; i++ {
		msg := makeRandomMessage()
		assert.NoError(backlog.Add(&msg))
		backlog.CancelRemove(&msg)
	}
	for i := 0; i < messageCount; i++ {
		msg := backlog.Get(true)
		backlog.CancelRemove(msg)
	}
	for i := 0; i < messageCount; i++ {
		msg := backlog.Get(true)
		backlog.ConfirmRemove(msg)
	}

	if backlog.Get(false) != nil {
		assert.Fail("Should not get another message")
	}

}

// Benchmark the add operation in the backlog
func BenchmarkBacklogAddNoConfirm(b *testing.B) {
	assert := require.New(b)
	defer os.Remove(backlogUDPDatabase)

	backlog, err := newMessageBacklog(backlogUDPDatabase)
	assert.NotNil(backlog)
	assert.NoError(err)
	// Preload with a few messages. Target is about 50-100k msg/sec so five seconds
	// outage should be roughly 500k messages.
	for i := 0; i < 500000; i++ {
		m := makeRandomMessage()
		backlog.Add(&m)
		backlog.CancelRemove(&m)
	}

	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		msg := makeRandomMessage()
		backlog.Add(&msg)
		backlog.CancelRemove(&msg)
	}

	b.StopTimer()
	d := float64(time.Since(start)) / float64(time.Second)

	b.ReportMetric(float64(b.N)/d, "op/sec")
}

// Benchmark the remove operation in the backlog
func BenchmarkBacklogAddConfirm(b *testing.B) {
	assert := require.New(b)
	defer os.Remove(backlogUDPDatabase)
	backlog, err := newMessageBacklog(backlogUDPDatabase)
	assert.NoError(err)
	assert.NotNil(backlog)

	// Preload with a few messages. Target is about 50-100k msg/sec so five seconds
	// outage should be roughly 500k messages.
	for i := 0; i < 500000; i++ {
		m := makeRandomMessage()
		backlog.Add(&m)
		backlog.CancelRemove(&m)
	}

	start := time.Now()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		msg := makeRandomMessage()
		backlog.Add(&msg)
		backlog.ConfirmRemove(&msg)
	}

	b.StopTimer()
	d := float64(time.Since(start)) / float64(time.Second)

	b.ReportMetric(float64(b.N)/d, "op/sec")

}
