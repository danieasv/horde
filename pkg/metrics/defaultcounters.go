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
// This is the default counters in the package. They include convenience
// methods. This *does* introduce pacakge-level state but there's a *lot* less
// clutter for the code while not introducing additional dependencies.
//
// It works similarly to the http.DefaultClient construct in the standard
// library.

// DefaultCoreCounters is the default core counters.
var DefaultCoreCounters *CoreCounters

// DefaultRADIUSCounters is the default RADIUS counters
var DefaultRADIUSCounters *RADIUSCounters

// DefaultAPNCounters is the default APN counters
var DefaultAPNCounters *APNCounters

// DefaultStoreCounters is the default data store counters
var DefaultStoreCounters *DataStoreCounters

func init() {

	DefaultCoreCounters = NewCoreCounters()
	DefaultRADIUSCounters = NewRADIUSCounters()
	DefaultAPNCounters = NewAPNCounters()
	DefaultStoreCounters = NewDataStoreCounters()
}
