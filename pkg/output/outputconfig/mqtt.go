package outputconfig

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
const (
	// MQTTEndpoint is the configuration key for the endpoint parameter in
	// the MQTT configuration
	MQTTEndpoint = "endpoint"
	// MQTTDisableCertCheck is the configuration key for the
	// disableCertCheck parameter in the MQTT configuration
	MQTTDisableCertCheck = "disableCertCheck"
	// MQTTUsername is the configuration key for the username parameter in
	// the MQTT configuration
	MQTTUsername = "username"
	// MQTTPassword is the configuration key for the password parameter in
	// the MQTT configuration
	MQTTPassword = "password"
	// MQTTClientID is the configuration key for the clientId parameter in
	// the MQTT configuration
	MQTTClientID = "clientId"
	// MQTTTopicName is the configuration key for the topicName parameter
	// in the MQTT configuration
	MQTTTopicName = "topicName"
)
