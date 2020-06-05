package utils
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
	"strings"
	"time"

	"github.com/ExploratoryEngineering/logging"
)

// LogParameters contains the default log parameters
type LogParameters struct {
	Level string `param:"desc=Logging level (debug, info, warning, error);default=debug;options=debug,info,warning,error"`
	Type  string `param:"desc=Log type (syslog, plain, plus, fancy);default=plain;options=syslog,plus,fancy,plain"`
}

var memoryLogs []*logging.MemoryLogger

// LogLevel returns the log level as a value
func logLevelFromString(level string) uint {
	if level == "" {
		return logging.DebugLevel
	}
	switch strings.ToLower(level)[0] {
	case 'd':
		return logging.DebugLevel
	case 'i':
		return logging.InfoLevel
	case 'w':
		return logging.WarningLevel
	case 'e':
		return logging.ErrorLevel
	default:
		return logging.WarningLevel
	}
}

const (
	plusLogs  = "plus"
	fancyLogs = "fancy"
	sysLogs   = "syslog"
)

// InitLogs configures logging for a service. This will turn on syslog
// logs  if they are enabled or just a plain text stderr log. If there
// are any errors while running
func InitLogs(service string, params LogParameters) {
	logging.SetLogLevel(logLevelFromString(params.Level))
	if params.Type == sysLogs {
		logging.EnableNamedSyslog(service)
		return
	}
	if params.Type == plusLogs {
		logging.EnableStderr(false)
		return
	}
	if params.Type == fancyLogs {
		memoryLogs = logging.NewMemoryLoggers(512)
		logging.EnableMemoryLogger(memoryLogs)
		go func() {
			tl := logging.NewTerminalLogger(memoryLogs)
			if err := tl.Start(); err != nil {
				logging.EnableStderr(true)
				logging.Error("Error launching terminal logger: %v", err)
				return
			}
			logging.EnableStderr(true)
			time.Sleep(300 * time.Millisecond)
			SendInterrupt()
			// Signal self to stop
		}()
		return
	}
	logging.EnableStderr(true)
}
