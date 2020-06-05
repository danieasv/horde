package output

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
	"sync/atomic"
	"time"

	"github.com/eesrc/horde/pkg/model"
)

// NullOutput is an output that will discard all received messages.
type nullOutput struct {
	started   bool
	terminate chan bool
	received  int32
}

func newNullOutput() Output {
	ret := nullOutput{}
	atomic.StoreInt32(&ret.received, 0)
	ret.terminate = make(chan bool)
	return &ret
}

func init() {
	registerOutput("null", newNullOutput)
}

func (n *nullOutput) messageReader(ch <-chan interface{}) {
	for {
		_, ok := <-ch
		if !ok {
			n.terminate <- true
			return
		}
		atomic.AddInt32(&n.received, 1)
	}
}

func (n *nullOutput) Validate(config model.OutputConfig) (model.ErrorMessage, error) {
	return nil, nil
}

func (n *nullOutput) Start(config model.OutputConfig, collectionFieldMask model.FieldMask, systemFieldMask model.FieldMask, message <-chan interface{}) {
	go n.messageReader(message)
	n.started = true
}

func (n *nullOutput) Stop(timeout time.Duration) {
	select {
	case <-n.terminate:
		break
	case <-time.After(timeout):
		break
	}
	n.started = false
}

func (n *nullOutput) Logs() []model.OutputLogEntry {
	var le []model.OutputLogEntry
	if n.started {
		le = append(le, model.OutputLogEntry{Message: "started", Time: time.Now(), Repeated: 0})
	}
	return le
}

func (n *nullOutput) Status() model.OutputStatus {
	return model.OutputStatus{
		Received:    int(atomic.LoadInt32(&n.received)),
		Forwarded:   0,
		ErrorCount:  0,
		Retransmits: 0,
	}
}
