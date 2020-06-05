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
	"io/ioutil"
	"log"
	"net/http"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Unable to read body: %v", err)
		return
	}
	log.Printf("Got: %s", string(body))
	w.WriteHeader(http.StatusOK)
}

func startWebserver(ep string) {
	handler := http.NewServeMux()
	handler.HandleFunc("/webhook", webhookHandler)
	if err := http.ListenAndServe(ep, handler); err != nil {
		log.Printf("Got error launching local web server: %v", err)
	}
}
func main() {
	startWebserver("127.0.0.1:9090")
}
