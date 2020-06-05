package restapi
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
	"encoding/json"
	"net/http"
)

func (s *restServer) rootHandler(w http.ResponseWriter, r *http.Request) {
	ret := struct {
		LoginURL            string `json:"login-url"`
		LogoutURL           string `json:"logout-url"`
		GHLoginURL          string `json:"github-login-url"`
		GHLogoutURL         string `json:"github-logout-url"`
		ProfileURL          string `json:"profile"`
		TeamURL             string `json:"teams"`
		DumpURL             string `json:"data-dump"`
		SystemURL           string `json:"system-defaults"`
		CollectionURL       string `json:"collections"`
		CollectionDetailURL string `json:"collection-detail"`
		DeviceURL           string `json:"device-list"`
		DeviceDetailURL     string `json:"device-detail"`
		OutputURL           string `json:"outputs"`
		OutputDetailURL     string `json:"output-detail"`
		FirmwareURL         string `json:"firmware-images"`
		FirmwareDetailURL   string `json:"firmware-detail"`
		FirmwareInUseURL    string `json:"firmware-image-in-use"`
	}{
		"/connect/login",
		"/connect/logout",
		"/github/login",
		"/github/logout",
		"/profile",
		"/teams",
		"/datadump",
		"/system",
		"/collections",
		"/collections/{id}",
		"/collections/{cid}/devices",
		"/collections/{cid}/devices/{did}",
		"/collections/{cid}/outputs",
		"/collections/{cid}/outputs/{oid}",
		"/collections/{cid}/firmware",
		"/collections/{cid}/firmware/{fid}",
		"/collections/{cid}/firmware/{fid}/usage",
	}
	json.NewEncoder(w).Encode(ret)
}
