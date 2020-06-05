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
	"time"

	"github.com/eesrc/horde/pkg/model"
)

// Logger is the in-memory logger used by the outputs. The logger keeps track
// of the log messages.
type Logger struct {
	entries []model.OutputLogEntry
	count   int
}

const maxEntries = 10

// NewLogger creates a new logger
func NewLogger() Logger {
	return Logger{make([]model.OutputLogEntry, maxEntries), 0}
}

// Append appends a new log entry
func (l *Logger) Append(msg string) {
	prevIndex := (l.count - 1) % maxEntries

	if l.count > 0 && l.entries[prevIndex].Message == msg {
		l.entries[prevIndex].Repeated++
		l.entries[prevIndex].Time = time.Now()
		return
	}
	newEntry := model.OutputLogEntry{Message: msg, Repeated: 0, Time: time.Now()}
	l.entries[l.count%maxEntries] = newEntry
	l.count++
}

// Messages returns the number of logged messages
func (l *Logger) Messages() int {
	return l.count
}

// Entries returns the log entries
func (l *Logger) Entries() []model.OutputLogEntry {
	if l.count < maxEntries {
		return l.entries[0:l.count]
	}
	ret := make([]model.OutputLogEntry, maxEntries)
	start := (l.count - maxEntries) % maxEntries
	for i := 0; i < maxEntries; i++ {
		ret[i] = l.entries[(start+i)%maxEntries]
	}
	return ret
}
