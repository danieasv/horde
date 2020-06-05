// Package threegpp contains the 3GPP RADIUS extensions.
//
//go:generate radius-dict-gen -package threegpp -output generated.go dictionary.3gpp
//
//
// Make sure you install the dictionary generator if you want to
// regenerate the generated.go file.  Install the dictionary generator
// like so:
//
//    go get -u layeh.com/radius/cmd/radius-dict-gen
//
package threegpp
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