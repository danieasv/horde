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
	"testing"
	"time"
)

type testCounterStore struct {
}

func (t *testCounterStore) Users() int64 {
	return 42
}
func (t *testCounterStore) Teams() int64 {
	return 42
}
func (t *testCounterStore) Devices() int64 {
	return 42
}
func (t *testCounterStore) Collections() int64 {
	return 42
}
func (t *testCounterStore) Outputs() int64 {
	return 42
}

func TestCoreCounters(t *testing.T) {
	cc := NewCoreCounters()
	cc.Start()
	cc.Update(&testCounterStore{})
	cc.AddHTTPStatus(200)
	cc.AddHTTPStatus(400)
	cc.AddHTTPStatus(500)
	cc.AddHTTPStatus(200)
	cc.AddHTTPResponseTime("GET", time.Millisecond*100)
}
