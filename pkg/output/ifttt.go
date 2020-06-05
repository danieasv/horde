package output

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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output/outputconfig"
	"github.com/eesrc/horde/pkg/utils/audit"
)

// ifttt is a special-purpose webhook integration with IFTTT
type ifttt struct {
	status model.OutputStatus
	logs   Logger
	config model.OutputConfig
	mutex  *sync.Mutex
	client *http.Client
}

// newIFTTT creates a new output.
func newIFTTT() Output {
	client := http.Client{Timeout: defaultHTTPClientTimeout}
	return &ifttt{
		mutex:  &sync.Mutex{},
		logs:   NewLogger(),
		client: &client,
	}
}

func init() {
	registerOutput("ifttt", newIFTTT)
}

// iftttBody is a temporary structure for messages
type iftttBody struct {
	Value1 string `json:"value1"`
	Value2 string `json:"value2"`
	Value3 string `json:"value3"`
}

func (i *ifttt) sendMessage(data iftttBody, event, key string) bool {
	buf, err := json.Marshal(&data)
	if err != nil {
		logging.Warning("Unable to marshal ifttt body: %v", err)
		return false
	}
	body := bytes.NewReader(buf)
	ifttURL := fmt.Sprintf("https://maker.ifttt.com/trigger/%s/with/key/%s", event, key)
	req, err := http.NewRequest("POST", ifttURL, body)
	if err != nil {
		logging.Warning("Unable to create request for webhook POST: %v", err)
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := i.client.Do(req)
	if err != nil {
		logging.Warning("Error calling IFTTT URL: %v", err)
		i.mutex.Lock()
		i.status.ErrorCount++
		i.mutex.Unlock()
		return false
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		i.logs.Append(fmt.Sprintf("Got %d response code from IFTTT", res.StatusCode))
		time.Sleep(5 * time.Second)
		i.mutex.Lock()
		i.status.ErrorCount++
		i.mutex.Unlock()
		return false
	}
	metrics.DefaultCoreCounters.MessagesForwardIFTTT.Add(1)
	return true
}

func (i *ifttt) sender(receiver <-chan interface{}, event, key string, asIs bool) {
	i.logs.Append("Starting IFTTT output")
	for msg := range receiver {
		dataMessage, ok := msg.(model.DataMessage)
		if !ok {
			logging.Debug("Output message isn't output data. Skipping")
			continue
		}
		i.mutex.Lock()
		i.status.Received++
		i.mutex.Unlock()
		var payload string
		if asIs {
			payload = string(dataMessage.Payload)
		} else {
			payload = base64.StdEncoding.EncodeToString(dataMessage.Payload)
		}
		data := iftttBody{Value1: payload, Value2: dataMessage.Device.ID.String(), Value3: ""}
		retries := 0
		success := false
		for retries < 3 && !success {
			success = i.sendMessage(data, event, key)
			retries++
			if retries > 1 {
				i.mutex.Lock()
				i.status.Retransmits++
				i.mutex.Unlock()
			}
		}
		i.mutex.Lock()
		i.status.Forwarded++
		audit.Log("IFTTT: Forwarded %d bytes from device with IMSI %d, Device ID=%s, Collection ID=%s",
			len(dataMessage.Payload), dataMessage.Device.IMSI,
			dataMessage.Device.ID.String(), dataMessage.Device.CollectionID.String())
		i.mutex.Unlock()
	}
	i.logs.Append("IFTTT output stopped")
}

func (i *ifttt) Validate(config model.OutputConfig) (model.ErrorMessage, error) {
	errs := validateConfig(config, []fieldSpec{
		fieldSpec{outputconfig.IFTTTEvent, reflect.String, true},
		fieldSpec{outputconfig.IFTTTKey, reflect.String, true},
		fieldSpec{outputconfig.FTTTAsIsPayload, reflect.Bool, false},
	})
	if len(errs) > 0 {
		return errs, errors.New("invalid config")
	}
	return errs, nil
}

func (i *ifttt) Start(config model.OutputConfig, collectionFieldMask model.FieldMask, systemFieldMask model.FieldMask, message <-chan interface{}) {
	// Note: Field mask is ignored at this point since it doesn't use any of the masked fields
	if _, err := i.Validate(config); err != nil {
		logging.Warning("Invalid config. Stopping output")

	}
	i.mutex.Lock()
	i.config = config
	i.mutex.Unlock()

	event := config[outputconfig.IFTTTEvent].(string)
	key := config[outputconfig.IFTTTKey].(string)
	tmp := config[outputconfig.FTTTAsIsPayload]
	asIs, ok := tmp.(bool)
	if !ok {
		asIs = false
	}
	go i.sender(message, event, key, asIs)
}

func (i *ifttt) Stop(timeout time.Duration) {
	// Do nothing. The channel will be closed and the output will be stopped automatically.
}

func (i *ifttt) Logs() []model.OutputLogEntry {
	return i.logs.Entries()
}

func (i *ifttt) Status() model.OutputStatus {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	ret := i.status
	return ret
}
