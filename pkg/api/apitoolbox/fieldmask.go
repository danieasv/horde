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
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
)

// SetFieldMask populates a model.FieldMask type with values from a
// apipb.FieldMask. It will return true if the model.FieldMask value changes.
func SetFieldMask(existing *model.FieldMask, new *apipb.FieldMask, system model.FieldMaskParameters) bool {
	if new == nil {
		return false
	}
	modified := *existing
	if new.Imei != nil {
		modified.Set(model.IMEIMask, system.ForcedFields(), new.Imei.Value)
	}
	if new.Imsi != nil {
		modified.Set(model.IMSIMask, system.ForcedFields(), new.Imsi.Value)
	}
	if new.Location != nil {
		modified.Set(model.LocationMask, system.ForcedFields(), new.Location.Value)
	}
	if new.Msisdn != nil {
		modified.Set(model.MSISDNMask, system.ForcedFields(), new.Msisdn.Value)
	}
	if modified != *existing {
		*existing = modified
		return true
	}
	return false
}
