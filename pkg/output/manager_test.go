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
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
)

const numOutputs = 10

func managerTest(t *testing.T, mgr Manager) {
	ms := sqlstore.NewMemoryStore()

	outputs := make([]model.Output, numOutputs)
	for i := range outputs {
		outputs[i].ID = ms.NewOutputID()
		outputs[i].Type = "null"
		outputs[i].Config = make(model.OutputConfig)
		outputs[i].Enabled = true
		strs, err := mgr.Verify(outputs[i])
		if err != nil {
			t.Fatal("Got error verifying output: ", err)
		}
		if len(strs) > 0 {
			t.Fatalf("Verification errors for output: %+v", strs)
		}
	}
	mgr.Refresh(outputs, 0)

	if _, err := mgr.Get(ms.NewOutputID()); err == nil {
		t.Fatal("Expected error when retrieving unknown output but didn't")
	}

	ops := make([]Output, numOutputs)
	for i := 0; i < len(outputs); i++ {
		var err error
		ops[i], err = mgr.Get(outputs[i].ID)
		if err != nil {
			t.Fatal("Got error retrieving output: ", err)
		}
	}

	for i := range outputs {
		if err := mgr.Update(outputs[i], 0); err != nil {
			t.Fatal("Unable to update output: ", err)
		}
	}
	// Retrieve the logs
	for i := 0; i < len(ops); i++ {
		ops[i].Logs()
		ops[i].Status()
		if err := mgr.Stop(outputs[i].ID); err != nil {
			t.Fatal("Unable to stop outputs")
		}
	}
	mgr.Refresh(outputs, 0)
	mgr.Shutdown()
}

func TestLocalManager(t *testing.T) {
	managerTest(t, NewLocalManager())
}
