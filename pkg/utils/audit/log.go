package audit

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
// This is an ad hoc package. We might need additional granularity later on
// for different kinds of audit logs but we'll keep this for now.

import (
	"sync/atomic"

	"github.com/ExploratoryEngineering/logging"
)

// Enable enables audit logging
func Enable() {
	atomic.StoreInt32(enabled, 1)
}

// Disable disables audit logging
func Disable() {
	atomic.StoreInt32(enabled, 0)
}

var enabled *int32

func init() {
	enabled = new(int32)
	atomic.StoreInt32(enabled, 0)
}

// Log writes an entry to the audit log
func Log(message string, args ...interface{}) {
	if atomic.LoadInt32(enabled) != 0 {
		logging.Info(message, args...)
	}
}
