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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/output/outputconfig"
	"github.com/eesrc/horde/pkg/utils/audit"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
)

// webhook is the output for web hooks, ie POSTs to random places on the
// interwebs. Each post contains one or more messages.
// Configuration for each output is quite simple: Just an URL with an optional
// basic auth. The web hooks will throttle back if the server returns anything
// but a 2xx status code. The first time it will throttle back 1 second, then
// 2 seconds, then 4 seconds until it reaches 256 seconds when it will be
// disabled.
// The webhook may either use a header with a secret or basic auth with
// an username and a password.
type webhook struct {
	terminate           chan bool
	status              model.OutputStatus
	logs                Logger
	config              model.OutputConfig
	mutex               *sync.Mutex
	nextSendTime        time.Time
	backOffTime         time.Duration
	client              *http.Client
	collectionFieldMask model.FieldMask
	systemFieldMask     model.FieldMask
}

const defaultHTTPClientTimeout = 10 * time.Second

func (w *webhook) configURL() string {
	val, ok := w.config[outputconfig.WebhookURLField]
	if !ok {
		return ""
	}
	return val.(string)
}

func (w *webhook) hasBasicAuth() bool {
	_, userExists := w.config[outputconfig.WebhookBasicAuthUser]
	_, passExists := w.config[outputconfig.WebhookBasicAuthPass]
	return userExists && passExists
}

func (w *webhook) configString(name string) string {
	v, ok := w.config[name]
	if !ok {
		return ""
	}
	return v.(string)
}

func (w *webhook) hasCustomHeader() bool {
	v1, hasName := w.config[outputconfig.WebhookCustomHeaderName]
	v2, hasValue := w.config[outputconfig.WebhookCustomHeaderValue]
	n, ok1 := v1.(string)
	v, ok2 := v2.(string)
	return hasName && hasValue && ok1 && ok2 && len(n) > 0 && len(v) > 0
}

// newWebhook creates a new output
func newWebhook() Output {
	client := http.Client{Timeout: defaultHTTPClientTimeout}
	return &webhook{
		terminate:   make(chan bool),
		mutex:       &sync.Mutex{},
		logs:        NewLogger(),
		backOffTime: time.Second,
		client:      &client,
	}
}

func init() {
	registerOutput("webhook", newWebhook)
}

// sendMessage sends aggregated messages to the configured endpoint.
func (w *webhook) sendMessages(msgs *apipb.ListMessagesResponse) bool {
	if time.Now().Before(w.nextSendTime) {
		return false
	}
	ma := apitoolbox.JSONMarshaler()
	buf, err := ma.MarshalToString(msgs)
	if err != nil {
		logging.Warning("Unable to marshal webhook body: %v", err)
		return false
	}
	body := bytes.NewReader([]byte(buf))
	req, err := http.NewRequest("POST", w.configURL(), body)
	if err != nil {
		logging.Warning("Unable to create request for webhook POST: %v", err)
	}
	if w.hasBasicAuth() {
		req.SetBasicAuth(w.configString(outputconfig.WebhookBasicAuthUser), w.configString(outputconfig.WebhookBasicAuthPass))
	}
	req.Header.Add("Content-Type", "application/json")

	if w.hasCustomHeader() {
		req.Header.Add(w.configString(outputconfig.WebhookCustomHeaderName), w.configString(outputconfig.WebhookCustomHeaderValue))
	}

	res, err := w.client.Do(req)
	if err != nil {
		w.nextSendTime = time.Now().Add(w.backOffTime)
		w.backOffTime *= 2
		logging.Warning("Error calling remote URL: %v. Backoff is %d seconds", err, w.backOffTime/time.Second)
		w.mutex.Lock()
		w.status.ErrorCount++
		w.mutex.Unlock()
		return false
	}

	// Read the entire response, then discard. This ensures the http.Client is
	// reused properly.
	io.Copy(ioutil.Discard, res.Body)
	res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		w.backOffTime *= 2
		w.logs.Append(fmt.Sprintf("Got %d response code from %s. Will retry in %d seconds",
			res.StatusCode, w.configURL(), w.backOffTime/time.Second))
		w.nextSendTime = time.Now().Add(w.backOffTime)
		logging.Debug("Got %d response code from %s. Will retry in %d seconds",
			res.StatusCode, w.configURL(), w.backOffTime/time.Second)
		w.mutex.Lock()
		w.status.ErrorCount++
		w.mutex.Unlock()
		return false
	}
	for _, msg := range msgs.Messages {
		audit.Log("Webhook: Forwarded %d bytes from device with IMSI %s, Device ID=%s, Collection ID=%s",
			len(msg.Payload), msg.Device.Imsi.Value,
			msg.Device.DeviceId.Value, msg.Device.CollectionId.Value)
	}

	w.backOffTime = time.Second
	w.mutex.Lock()
	w.status.Forwarded += len(msgs.Messages)
	metrics.DefaultCoreCounters.MessagesForwardWebhook.Add(float64(len(msgs.Messages)))
	w.mutex.Unlock()

	return true
}

func (w *webhook) webhookSender(receiver <-chan interface{}) {
	if _, err := w.Validate(w.config); err != nil {
		w.logs.Append("Invalid configuration. Stopped.")
		return
	}
	for {
		select {
		case <-w.terminate:
			logging.Debug("terminate signal, webhook terminates")
			return
		case msg, ok := <-receiver:
			if !ok {
				return
			}
			msgs := &apipb.ListMessagesResponse{
				Messages: make([]*apipb.OutputDataMessage, 0),
			}
			for {
				if m, ok := msg.(model.DataMessage); ok {
					//TODO(stalehd): This should *probably* be the correct collection but
					// we'll save us a lookup at this point. The field mask and
					// the firmware settings are used to infer the firmware status
					// that should be set on the resulting device.
					tmpColl := model.NewCollection()
					tmpColl.FieldMask = w.collectionFieldMask
					msgs.Messages = append(msgs.Messages, apitoolbox.NewOutputDataMessageFromModel(m, tmpColl))
				} else {
					logging.Warning("Not a message: %T", m)
				}
				if len(receiver) == 0 {
					break
				}
				msg = <-receiver
			}
			w.mutex.Lock()
			w.status.Received += len(msgs.Messages)
			w.mutex.Unlock()

			if !w.sendMessages(msgs) {
				logging.Warning("Lost %d messages", len(msgs.Messages))
			}
		}
	}
}

func (w *webhook) Validate(config model.OutputConfig) (model.ErrorMessage, error) {
	errs := validateConfig(config, []fieldSpec{
		fieldSpec{outputconfig.WebhookURLField, reflect.String, true},
		fieldSpec{outputconfig.WebhookBasicAuthUser, reflect.String, false},
		fieldSpec{outputconfig.WebhookBasicAuthPass, reflect.String, false},
		fieldSpec{outputconfig.WebhookCustomHeaderName, reflect.String, false},
		fieldSpec{outputconfig.WebhookCustomHeaderValue, reflect.String, false},
	})
	val, ok := config[outputconfig.WebhookURLField]
	if ok {
		url, ok := val.(string)
		if ok {
			check := newEndpointChecker(url)
			if !check.IsValidHTTPURL() {
				errs[outputconfig.WebhookURLField] = "Invalid HTTP URL"
			}
			if !check.IsValidHost() {
				errs[outputconfig.WebhookURLField] = "Invalid host name"
			}
		}
	}
	if len(errs) > 0 {
		return errs, errors.New("invalid config")
	}
	return errs, nil
}

func (w *webhook) Start(config model.OutputConfig, collectionFieldMask model.FieldMask, systemFieldMask model.FieldMask, message <-chan interface{}) {
	w.mutex.Lock()
	w.collectionFieldMask = collectionFieldMask
	w.systemFieldMask = systemFieldMask
	w.config = config
	w.mutex.Unlock()
	go w.webhookSender(message)
}

func (w *webhook) Stop(timeout time.Duration) {
	select {
	case w.terminate <- true:
	default:
		// already terminated

	}
}

func (w *webhook) Logs() []model.OutputLogEntry {
	return w.logs.Entries()
}

func (w *webhook) Status() model.OutputStatus {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	ret := w.status
	ret.ErrorCount = w.logs.Messages()
	return ret
}
