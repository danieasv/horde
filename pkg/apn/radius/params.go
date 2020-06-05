package radius

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

// ServerParameters holds the configuration for the RADIUS server
type ServerParameters struct {
	Endpoint     string `param:"desc=RADIUS Authentication listen endpoint;default=localhost:1812"`
	SharedSecret string `param:"desc=RADIUS shared secret;default=radiussharedsecret"`
}
