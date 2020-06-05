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
	"fmt"
	"sync"

	"github.com/ExploratoryEngineering/logging"
)

type outputGenerator func() Output

var outputTypes map[string]outputGenerator
var mutex = &sync.Mutex{}

func registerOutput(id string, generator outputGenerator) {
	mutex.Lock()
	defer mutex.Unlock()
	if outputTypes == nil {
		outputTypes = make(map[string]outputGenerator)
	}
	outputTypes[id] = generator
}

func listOutputTypes() {
	mutex.Lock()
	defer mutex.Unlock()
	for k := range outputTypes {
		logging.Info("Output type registered: %s", k)
	}
}

func makeOutput(id string) (Output, error) {
	mutex.Lock()
	defer mutex.Unlock()

	ret, ok := outputTypes[id]
	if !ok {
		return nil, fmt.Errorf("unknown output type: %s", id)
	}
	return ret(), nil
}
