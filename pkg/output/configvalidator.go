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
	"reflect"

	"github.com/eesrc/horde/pkg/model"
)

type fieldSpec struct {
	name      string
	fieldType reflect.Kind
	required  bool
}

func validateConfig(config model.OutputConfig, fields []fieldSpec) model.ErrorMessage {
	errs := make(model.ErrorMessage)
	for _, v := range fields {
		exists, okType := config.HasParameterOfType(v.name, v.fieldType)
		if !exists && v.required {
			errs[v.name] = "required parameter is missing"
			continue
		}
		if exists && !okType {
			errs[v.name] = "parameter is incorrect type"
		}
	}
	return errs
}
