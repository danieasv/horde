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
)

func TestLogger(t *testing.T) {
	logger := NewLogger()

	if e := logger.Entries(); len(e) != 0 || logger.Messages() != 0 {
		t.Fatalf("Expected 0 elements but got %v", len(logger.Entries()))
	}
	logger.Append("1")
	logger.Append("2")
	logger.Append("2")
	logger.Append("3")
	logger.Append("4")
	logger.Append("4")
	logger.Append("4")
	logger.Append("4")
	logger.Append("4")

	le := logger.Entries()
	if len(le) != 4 || logger.Messages() != 4 {
		t.Fatalf("Incorrect number of log entries. Expected 4 got %v (%+v)", len(le), le)
	}
	if le[0].Message != "1" && le[0].Repeated == 0 {
		t.Fatalf("Expected '1' once got %v", le[0])
	}
	if le[1].Message != "2" && le[1].Repeated != 1 {
		t.Fatalf("Expected '2' twice got %v", le[1])
	}
	if le[2].Message != "3" && le[2].Repeated != 0 {
		t.Fatalf("Expected '3' once got %v", le[1])
	}
	if le[3].Message != "4" && le[3].Repeated != 4 {
		t.Fatalf("Expected '4' five times got %v", le[1])
	}

	logger.Append("5")
	logger.Append("6")
	logger.Append("7")
	logger.Append("8")
	logger.Append("9")
	logger.Append("9")
	logger.Append("9")
	logger.Append("9")
	logger.Append("10")
	if le := logger.Entries(); len(le) != 10 || logger.Messages() != 10 {
		t.Fatalf("Expected 10 entries, got %v", len(le))
	}
	logger.Append("11")
	logger.Append("12")
	logger.Append("13")
	logger.Append("13")
	logger.Append("13")
	logger.Append("13")
	logger.Append("13")
	logger.Append("14")
	logger.Append("15")

	// Output should be 6,7,8,9,10,11,12,13,14,15 in that order
	le = logger.Entries()
	check := func(exp string, val model.OutputLogEntry) {
		if val.Message != exp {
			t.Fatalf("Expected %v but got %v", exp, val.Message)
		}
	}
	check("6", le[0])
	check("7", le[1])
	check("8", le[2])
	check("9", le[3])
	check("10", le[4])
	check("11", le[5])
	check("12", le[6])
	check("13", le[7])
	check("14", le[8])
	check("15", le[9])
}
