package deviceio

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
import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNASList(t *testing.T) {
	assert := require.New(t)

	uparam := UDPParameters{
		NASID: "0,1,2",
	}
	cparam := CoAPParameters{
		NASID: "2,1,0",
	}
	l1 := uparam.NASList()
	l2 := cparam.NASList()

	assert.Contains(l1, 0, 1, 2)
	assert.Contains(l2, 0, 1, 2)

	uparam = UDPParameters{NASID: ""}
	assert.Len(uparam.NASList(), 0)
	cparam = CoAPParameters{NASID: "a,0,b,1"}

	assert.Len(cparam.NASList(), 2)
	assert.Contains(cparam.NASList(), 0, 1)
}
