package restapi

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
	"bufio"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/eesrc/horde/pkg/metrics"
)

// This is the status logger that logs the response code for requests going
// through the http server. WriteHeader is called if the status code <> 200
// the status field will hold the status code when the request is done.
type statusLogger struct {
	writer http.ResponseWriter
	status int
}

func (s *statusLogger) Header() http.Header {
	return s.writer.Header()
}

func (s *statusLogger) Write(b []byte) (int, error) {
	return s.writer.Write(b)
}

func (s *statusLogger) WriteHeader(status int) {
	s.writer.WriteHeader(status)
	s.status = status
}

// the http.Hijacker interface is used by the websockets
func (s *statusLogger) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := s.writer.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("writer isn't capable of hijacking")
	}
	return hj.Hijack()
}

func makeStatusLogger(w http.ResponseWriter) *statusLogger {
	return &statusLogger{writer: w, status: http.StatusOK}
}

func (s *restServer) perfCounterHandler(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusLogger := makeStatusLogger(w)
		start := time.Now()
		f(statusLogger, r)
		metrics.DefaultCoreCounters.AddHTTPStatus(statusLogger.status)
		metrics.DefaultCoreCounters.AddHTTPResponseTime(r.Method, time.Since(start))
	}
}
