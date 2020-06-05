package metrics

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
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// DataStoreCounters holds the counters for the data store
type DataStoreCounters struct {
	Stored prometheus.Counter
	Errors prometheus.Counter
}

// NewDataStoreCounters creates data store counters
func NewDataStoreCounters() *DataStoreCounters {
	ret := &DataStoreCounters{
		Stored: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "messages_stored",
			Help: "Number of messages stored in data store",
		}),
		Errors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "message_store_error",
			Help: "Number of store errors",
		}),
	}
	return ret
}

var datastoreInitCounters sync.Once

// Start registers the counters in prometheus
func (d *DataStoreCounters) Start() {
	datastoreInitCounters.Do(func() {
		prometheus.MustRegister(d.Stored)
		prometheus.MustRegister(d.Errors)
	})
	d.Stored.Add(0)
	d.Errors.Add(0)
}
