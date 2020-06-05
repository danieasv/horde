package ctrlh

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
	"fmt"

	"github.com/eesrc/horde/pkg/model"
)

// UtilCommand is the Kong command struct for the util subcommand
type UtilCommand struct {
	ID idConversionCommand `kong:"cmd,help='Decode API identifiers to internal identifiers'"`
	DI diConversionCommand `kong:"cmd,help='Encode internal identifiers into API identifiers'"`
}

type idConversionCommand struct {
	InternalIdentifiers []int64 `kong:"arg,help='IDs to decode'"`
}

func (c *idConversionCommand) Run(rc RunContext) error {
	for _, n := range rc.HordeCommands().Util.ID.InternalIdentifiers {
		k := model.DeviceKey(n)
		fmt.Printf("%d -> %s\n", n, k.String())
	}
	return nil
}

type diConversionCommand struct {
	APIIdentifiers []string `kong:"arg,help='API identifiers to decode'"`
}

func (c *diConversionCommand) Run(rc RunContext) error {
	for _, s := range rc.HordeCommands().Util.DI.APIIdentifiers {
		k, err := model.NewDeviceKeyFromString(s)
		if err != nil {
			fmt.Printf("*** %s is not a valid key", s)
			continue
		}
		fmt.Printf("%s -> %d\n", k.String(), int64(k))

	}
	return nil
}
