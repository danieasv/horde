package version

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

import "fmt"

// Number holds the version number for Horde. This is set at compile time
var Number = "0.0.0"

// CommitHash holds the commit hash for Horde. This is set at compile time
var CommitHash = "0000000000000000000"

// Name holds the code name for the version. This is set at compile time.
var Name = "develop"

// BuildDate holds the build date for Horde. This is set at compile time.
var BuildDate = "1970-01-01T00:00"

// Release returns the version string for Horde
func Release() string {
	return fmt.Sprintf("%s - %s (%s) %s", Number, Name, CommitHash[:6], BuildDate)
}
