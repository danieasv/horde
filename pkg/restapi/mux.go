package restapi

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
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/status"
)

// Create a custom handler and mux for the service. Root handler, Web sockets
// and file uploads are handled by regular handlers that wrap the gRPC services
// and all other requests are handled by the grpc-gateway mux

// MaxUploadSize is the maximum request body. Max body size is 2MB. This might
// be too small for the largest firmware images but it's a starting point.
// TODO(stalehd): This is not the final resting location for this constant.
const maxUploadSize = 2 * 1024 * 1024

func (s *restServer) createWrappedMux(mux *runtime.ServeMux) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			s.rootHandler(w, r)
			return
		}
		pathParts := strings.Split(r.URL.Path, "/")

		// Match
		// - /collections/{id}/from
		// -  /collections/{id}/firmware
		if len(pathParts) == 4 && pathParts[1] == "collections" {
			collectionID := pathParts[2]
			if pathParts[3] == "from" {
				newWebsocketHandler(collectionID, "", s.apiServer)(w, r)
				return
			}
			if pathParts[3] == "firmware" && r.Method == http.MethodPost {
				// This might be a www-form-multipart POST request. See if it
				// parses the request before forwarding to the wrapped handler
				err := r.ParseMultipartForm(maxUploadSize)
				if err == nil {
					// Parse OK which means we have a firmware upload. Forward
					// to the upload handler
					s.imageUploadHandler(collectionID, s.apiServer, w, r)
					return
				}
			}
		}
		// Matching the /collections/{id}/devices/{id}/from
		if len(pathParts) == 6 && pathParts[1] == "collections" && pathParts[3] == "devices" && pathParts[5] == "from" {
			collectionID := pathParts[2]
			deviceID := pathParts[4]
			newWebsocketHandler(collectionID, deviceID, s.apiServer)(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	}
}

func (s *restServer) imageUploadHandler(collectionID string, firmwareService apipb.HordeServer, w http.ResponseWriter, r *http.Request) {
	if firmwareService == nil {
		reportError(w, http.StatusInternalServerError, "No firmware upload available", nil)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		logging.Warning("Got error reading firmware image file for user: %v", err)
		reportError(w, http.StatusInternalServerError, "Unable to read firmware image", nil)
		return
	}
	defer file.Close()
	logging.Debug("Adding firmware image filename=%s length=%d", header.Filename, header.Size)
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		logging.Warning("Got error reading firmware image buffer: %v", err)
		reportError(w, http.StatusInternalServerError, "Unable to read firmware image buffer", nil)
		return
	}
	req := &apipb.CreateFirmwareRequest{
		CollectionId: &wrappers.StringValue{Value: collectionID},
		Image:        buf,
		Filename:     &wrappers.StringValue{Value: header.Filename},
	}
	fw, err := firmwareService.CreateFirmware(r.Context(), req)
	if err != nil {
		errorCode := runtime.HTTPStatusFromCode(status.Code(err))
		reportError(w, errorCode, err.Error(), nil)
		return
	}
	ma := apitoolbox.JSONMarshaler()
	str, err := ma.MarshalToString(fw)
	if err != nil {
		logging.Warning("Unable to marshal firmware JSON: %v", err)
		reportError(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(str))
}
