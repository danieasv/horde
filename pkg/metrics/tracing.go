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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime/trace"
	"strconv"
	"time"

	"github.com/ExploratoryEngineering/logging"
)

//
// Tracing endpoint. The trace is controlled via an unbuffered channel of
// time.Duration values. Each value is read off the channel and a trace is
// started with the given duration. The channel will block writing while a
// trace is running and reading is blocked until someone sends something on
// the trace channel.
//
var traceChan chan time.Duration

// EnableTracing starts the tracing goroutine
func enableTracingRoutine() {
	if traceChan != nil {
		return
	}
	traceChan = make(chan time.Duration)
	go func() {
		for duration := range traceChan {
			traceFileName := time.Now().Format("trace_2006-01-02T150405.out")
			traceFile, err := os.Create(traceFileName)
			if err != nil {
				logging.Error("Unable to create trace file '%s': %v", traceFileName, err)
				continue
			}
			logging.Info("Trace started for %d seconds. Trace file name is %s", int(duration.Seconds()), traceFileName)
			if err := trace.Start(traceFile); err != nil {
				logging.Error("Unable to start the trace: %v", err)
				traceFile.Close()
				continue
			}
			time.Sleep(duration)

			trace.Stop()
			traceFile.Close()
			logging.Info("Trace is completed. Results are placed in %s", traceFileName)
		}
	}()
}

// TraceHandler is a simple http.HandleFunc that handles POST requests. A new
// trace is started with there's a POST request and the trace channel isn't
// blocking. If the trace channel is blocking 409 conflict will be returned.
// All other methods returns 405 method not allowed.
func traceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			buf, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Specify time to trace in body", http.StatusBadRequest)
				return
			}
			seconds, err := strconv.Atoi(string(buf))
			if err != nil || seconds < 1 {
				http.Error(w, "Specify number of seconds in request body", http.StatusBadRequest)
				return
			}
			select {
			case traceChan <- time.Second * time.Duration(seconds):
				io.WriteString(w, "Trace started")
			default:
				http.Error(w, "Trace in progress", http.StatusConflict)
			}
		default:
			http.Error(w, "Illegal method", http.StatusMethodNotAllowed)
		}
	}
}
