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
	"crypto/tls"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/eesrc/horde/pkg/api/apitoolbox"
	"github.com/eesrc/horde/pkg/metrics"
	"github.com/eesrc/horde/pkg/output/outputconfig"
	"github.com/eesrc/horde/pkg/utils/audit"

	"github.com/ExploratoryEngineering/logging"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/eesrc/horde/pkg/model"
)

type mqttConfig struct {
	endpoint         string
	disableCertCheck bool
	username         string
	password         string
	clientID         string
	topicName        string
}

func init() {
	registerOutput("mqtt", newMQTT)
}

func newMQTTConfig(config model.OutputConfig) mqttConfig {
	ret := mqttConfig{}
	val, ok := config[outputconfig.MQTTEndpoint]
	if ok {
		ret.endpoint, _ = val.(string)
	}
	val, ok = config[outputconfig.MQTTDisableCertCheck]
	if ok {
		ret.disableCertCheck, _ = val.(bool)
	}
	val, ok = config[outputconfig.MQTTUsername]
	if ok {
		ret.username, _ = val.(string)
	}
	val, ok = config[outputconfig.MQTTPassword]
	if ok {
		ret.password, _ = val.(string)
	}
	val, ok = config[outputconfig.MQTTClientID]
	if ok {
		ret.clientID, _ = val.(string)
	}
	val, ok = config[outputconfig.MQTTTopicName]
	if ok {
		ret.topicName, _ = val.(string)
	}
	return ret
}

type mqttOutput struct {
	client              mqtt.Client
	logs                Logger
	status              model.OutputStatus
	mutex               sync.Mutex
	collectionFieldMask model.FieldMask
	systemFieldMask     model.FieldMask
}

func newMQTT() Output {
	return &mqttOutput{logs: NewLogger()}
}

// Validate verifies the configuraton for the output. Error messages that
// should be provided to the end user is returned as the 2nd parameter.
// On success the error return value is nil.
func (m *mqttOutput) Validate(config model.OutputConfig) (model.ErrorMessage, error) {
	errs := validateConfig(config, []fieldSpec{
		fieldSpec{outputconfig.MQTTEndpoint, reflect.String, true},
		fieldSpec{outputconfig.MQTTClientID, reflect.String, true},
		fieldSpec{outputconfig.MQTTTopicName, reflect.String, true},
		fieldSpec{outputconfig.MQTTDisableCertCheck, reflect.Bool, false},
		fieldSpec{outputconfig.MQTTPassword, reflect.String, false},
		fieldSpec{outputconfig.MQTTUsername, reflect.String, false},
	})
	val, ok := config[outputconfig.MQTTEndpoint]
	if ok {
		ep, ok := val.(string)
		if ok {
			check := newEndpointChecker(ep)
			if !check.IsValidMQTTEndpoint() {
				errs[outputconfig.MQTTEndpoint] = "Invalid endpoint URL. Must include protocol, port and no path."
			}
			if !check.IsValidHost() {
				errs[outputconfig.MQTTEndpoint] = "Unknown or invalid host name"
			}
		}
	}
	if len(errs) > 0 {
		return errs, errors.New("invalid config")
	}
	return errs, nil
}

// Start launches the output. This is non blocking, ie if the output
// must connect to a remote server or perform some sort initialization it
// will do so in a separate goroutine. The output will stop automatically
// when the message channel is closed. If the message channel is closed
// the output should attempt empty any remaining messages in the queue.
func (m *mqttOutput) Start(config model.OutputConfig, collectionFieldMask model.FieldMask, systemFieldMask model.FieldMask, messages <-chan interface{}) {
	m.collectionFieldMask = collectionFieldMask
	m.systemFieldMask = systemFieldMask
	if _, err := m.Validate(config); err != nil {
		logging.Warning("Invalid config, won't start MQTT output: %+v", config)
	}
	mqttConf := newMQTTConfig(config)
	checker := newEndpointChecker(mqttConf.endpoint)
	opts := mqtt.NewClientOptions()
	logging.Debug("Starting MQTT broker with config %+v and field mask %b", mqttConf, m.collectionFieldMask)
	opts.AddBroker(mqttConf.endpoint)
	opts.SetClientID(mqttConf.clientID)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)
	opts.SetWriteTimeout(1 * time.Second)
	// If the client auto reconnects it will block until a connection becomes
	// available. This isn't very helpful if the messages is going to be
	// queued and the decoding pipeline might drop messages that aren't processed
	// quickly enough by the clients. The backlog will keep up to 50 messages
	// in memory until they are discarded.
	opts.SetAutoReconnect(false)
	opts.SetMessageChannelDepth(1)
	opts.SetCleanSession(true)
	if mqttConf.username != "" {
		opts.SetUsername(mqttConf.username)
	}
	if mqttConf.password != "" {
		opts.SetPassword(mqttConf.password)
	}

	if checker.IsSSLScheme() {
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: mqttConf.disableCertCheck,
		})
	}

	m.client = mqtt.NewClient(opts)

	go m.sender(messages, mqttConf)
}

func (m *mqttOutput) connect() {
	token := m.client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		m.logs.Append(fmt.Sprintf("Unable to connect to broker: %v", err.Error()))
	}
	m.logs.Append("Connected to broker")
}

func (m *mqttOutput) sender(messages <-chan interface{}, config mqttConfig) {
	if m.client == nil {
		logging.Warning("MQTT client is nil. Terminating output to %s", config.endpoint)
		return
	}
	for msg := range messages {
		logging.Debug("MQTT received message from device -> %s", config.endpoint)
		if !m.client.IsConnected() {
			// Attempt a reconnect
			m.connect()
		}
		qos := byte(1)
		retained := false

		m.mutex.Lock()
		m.status.Received++
		m.mutex.Unlock()
		// Payload is an data structure. Convert into same format as the websocket
		// output (apiDeviceData) and pass on.
		dataMsg, ok := msg.(model.DataMessage)
		if !ok {
			logging.Warning("Didn't receive a DataMessage type on channel but got %T. Silently dropping it.", msg)
			continue
		}
		logging.Info("Making new output from model message. Field mask = %b", m.collectionFieldMask)
		tmpColl := model.NewCollection()
		tmpColl.FieldMask = m.collectionFieldMask
		dataOutput := apitoolbox.NewOutputDataMessageFromModel(dataMsg, tmpColl)
		ma := apitoolbox.JSONMarshaler()
		str, err := ma.MarshalToString(dataOutput)
		if err != nil {
			logging.Warning("Unable to marshal %T into JSON: %v. Silently dropping it.", dataMsg, err)
			continue
		}
		token := m.client.Publish(config.topicName, qos, retained, []byte(str))
		token.Wait()
		if err := token.Error(); err != nil {
			logging.Info("Unable to send message to MQTT server %s: %v", config.endpoint, err)
			m.logs.Append(fmt.Sprintf("Error sending message: %s", err.Error()))
			m.mutex.Lock()
			m.status.ErrorCount++
			m.mutex.Unlock()
			continue
		}
		m.mutex.Lock()
		m.status.Forwarded++
		m.mutex.Unlock()
		metrics.DefaultCoreCounters.MessagesForwardMQTT.Add(1)
		audit.Log("MQTT: Forwarded %d bytes from device with IMSI %d, Device ID=%s, Collection ID=%s",
			len(dataMsg.Payload), dataMsg.Device.IMSI,
			dataMsg.Device.ID.String(), dataMsg.Device.CollectionID.String())
	}
	logging.Debug("Output channel closed for MQTT output to %s/%s", config.endpoint, config.topicName)
}

// Stop halts the output. Any buffered messages that can't be sent during
// the timeout will be discarded by the output. When the Stop call returns
// the output has stopped.
func (m *mqttOutput) Stop(timeout time.Duration) {
	if m.client == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			logging.Warning("Recovered from panic: %v", r)
		}
	}()
	m.client.Disconnect(250)
	m.logs.Append("Disconnected from broker")
}

// Logs returns end user logs for the output.
func (m *mqttOutput) Logs() []model.OutputLogEntry {
	return m.logs.Entries()
}

// Status reports the internal status of the forwarder.
func (m *mqttOutput) Status() model.OutputStatus {
	return m.status
}
