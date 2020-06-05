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
	"context"
	"encoding/json"
	"net/http"

	"github.com/ExploratoryEngineering/logging"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/status"
)

// The error message
type httpError struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Info    interface{} `json:"additionalInfo,omitempty"`
}

// reportError reports an error to the http client
func reportError(w http.ResponseWriter, status int, message string, info interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(httpError{status, message, info})
}

// CustomHTTPError is the custom http error handler for grpc-gateway.
// The default handler leaks implementation details so it's not ideal.
func CustomHTTPError(ctx context.Context, _ *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-type", marshaler.ContentType())
	ret := httpError{Status: 500, Message: "Error"}
	s, ok := status.FromError(err)
	if ok {
		ret.Status = runtime.HTTPStatusFromCode(s.Code())
		ret.Message = s.Message()
		ret.Info = s.Details()
		logging.Debug("grpc error: code: %s message: %s details: %+v, request URI=%s", s.Code().String(), s.Message(), s.Details(), r.RequestURI)
	}
	w.WriteHeader(ret.Status)
	jsonErr := json.NewEncoder(w).Encode(ret)
	if jsonErr != nil {
		w.Write([]byte(`{"message":"Error", "status":500}`))
	}
}
