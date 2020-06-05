package apitoolbox

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

	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
)

func TestSetFieldMask(t *testing.T) {
	assert := require.New(t)

	fm := model.FieldMaskParameters{
		Default: "imsi,msisdn",
		Forced:  "msisdn",
	}

	// Assign nil value should return false, ie no change
	var existing model.FieldMask = fm.DefaultFields()
	assert.False(SetFieldMask(&existing, nil, fm))

	// Empty parameter should also return false
	assert.False(SetFieldMask(&existing, &apipb.FieldMask{}, fm))

	// The defaults won't change
	assert.False(SetFieldMask(&existing, &apipb.FieldMask{
		Imei: &wrappers.BoolValue{Value: false},
	}, fm))
	assert.Equal(fm.DefaultFields(), existing)

	// Change the defaults should return true
	assert.True(SetFieldMask(&existing, &apipb.FieldMask{
		Location: &wrappers.BoolValue{Value: false},
		Imei:     &wrappers.BoolValue{Value: false},
		Imsi:     &wrappers.BoolValue{Value: false},
		Msisdn:   &wrappers.BoolValue{Value: false},
	}, fm))
	// ...but the forced fields will still be set
	assert.Equal(fm.ForcedFields(), existing)

	// ..and a stricter mask will return true
	assert.True(SetFieldMask(&existing, &apipb.FieldMask{
		Location: &wrappers.BoolValue{Value: false},
		Imei:     &wrappers.BoolValue{Value: true},
		Imsi:     &wrappers.BoolValue{Value: true},
		Msisdn:   &wrappers.BoolValue{Value: true},
	}, fm))
	assert.Equal(model.IMEIMask|model.IMSIMask|model.MSISDNMask, existing)
}
