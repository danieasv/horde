package main

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
	"fmt"
	"os"

	"github.com/eesrc/horde/pkg/htest"
	"github.com/eesrc/horde/pkg/storage"

	"github.com/ExploratoryEngineering/logging"
	"github.com/ExploratoryEngineering/params"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
)

type args struct {
	DB        sqlstore.Parameters
	UserCount int `param:"desc=Number of users to create;default=1000"`
}

func main() {
	var cfg args
	var store storage.DataStore
	var err error

	logging.SetLogLevel(logging.DebugLevel)
	if err = params.NewEnvFlag(&cfg, os.Args[1:]); err != nil {
		fmt.Println(err.Error())
		return
	}
	settings := htest.DefaultGeneratorSettings
	settings.DevicesPerCollection = 1000
	settings.LogActivity = true

	fmt.Println("Data about to be generated:")
	fmt.Printf("  Users:       %-8d\n", cfg.UserCount)
	fmt.Printf("  Tokens:      %-8d\n", htest.TokenCount(cfg.UserCount, settings))
	fmt.Printf("  Teams:       %-8d\n", htest.TeamCount(cfg.UserCount, settings))
	fmt.Printf("  Members:     %-8d\n", htest.MemberCount(cfg.UserCount, settings))
	fmt.Printf("  Collections: %-8d\n", htest.CollectionCount(cfg.UserCount, settings))
	fmt.Printf("  Outputs:     %-8d\n", htest.OutputCount(cfg.UserCount, settings))
	fmt.Printf("  Devices:     %-8d\n", htest.DeviceCount(cfg.UserCount, settings))

	fmt.Println("About to create lots of test data! Press <ENTER> to continue")

	r := bufio.NewReader(os.Stdin)
	r.ReadLine()

	store, err = sqlstore.NewSQLStore(cfg.DB.Type, cfg.DB.ConnectionString, cfg.DB.CreateSchema, 1, 1)
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := htest.Generate(cfg.UserCount, settings, store); err != nil {
		fmt.Println(err)
	}

	logging.Info("Users generated: %d", cfg.UserCount)
	logging.Info("Tokens generated: %d", htest.TokenCount(cfg.UserCount, settings))
	logging.Info("Teams generated: %d", htest.TeamCount(cfg.UserCount, settings))
	logging.Info("Members generated: %d", htest.MemberCount(cfg.UserCount, settings))
	logging.Info("Collections generated: %d", htest.CollectionCount(cfg.UserCount, settings))
	logging.Info("Outputs generated: %d", htest.OutputCount(cfg.UserCount, settings))
	logging.Info("Devices generated: %d", htest.DeviceCount(cfg.UserCount, settings))
	logging.Info("Completed.")
}
