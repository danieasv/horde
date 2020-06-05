package metrics

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
	"sync"

	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/prometheus/client_golang/prometheus"
)

// APNCounters contains counters for APN. The APN might contain more than
// one RADIUS server (currently it doesn't but that might change)
type APNCounters struct {
	MessagesReceived  *prometheus.CounterVec
	MessagesSent      *prometheus.CounterVec
	MessageSendErrors *prometheus.CounterVec
	MessagesForwarded *prometheus.CounterVec
	MessagesRejected  *prometheus.CounterVec
	MessagesError     prometheus.Counter
	Incoming          *prometheus.CounterVec
	Outgoing          *prometheus.CounterVec
	RequestRejected   prometheus.Counter
}

// NewAPNCounters creates a new set of counters for an APN
func NewAPNCounters() *APNCounters {
	ret := &APNCounters{
		MessagesReceived: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "apn_messages_received",
			Help: "Number of messages received by APN"},
			[]string{"apn"}),
		MessagesSent: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "apn_messages_sent",
			Help: "Number of messages sent via APN",
		}, []string{"apn"}),
		MessageSendErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "apn_messages_send_error",
			Help: "Errors for messages sent via APN",
		}, []string{"apn"}),
		MessagesForwarded: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "apn_messages_forwarded",
			Help: "Messages forwarded by the APN to Horde",
		}, []string{"apn"}),
		MessagesRejected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "apn_messages_rejected",
			Help: "Messages rejected by the APN",
		}, []string{"apn"}),
		MessagesError: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "apn_messages_error",
			Help: "Lookup errors for messages"}),
		Incoming: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "horde_incoming",
			Help: "Number of messages received (apn, type). Includes also FOTA",
		}, []string{"apn", "nas", "transport"}),
		Outgoing: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "horde_outgoing",
			Help: "Number of messages received (apn, type). Includes also FOTA",
		}, []string{"apn", "nas", "transport"}),
		RequestRejected: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "horde_upstream_rejected",
			Help: "Rejected gRPC upstream messages. This might be an incorrectly configured listener.",
		}),
	}
	return ret
}

var apnInitCounters sync.Once

// Start registers the counters.
func (a *APNCounters) Start(apnConfig *storage.APNConfigCache) {
	apnInitCounters.Do(func() {
		prometheus.MustRegister(a.MessagesReceived)
		prometheus.MustRegister(a.MessagesSent)
		prometheus.MustRegister(a.MessageSendErrors)
		prometheus.MustRegister(a.MessagesForwarded)
		prometheus.MustRegister(a.MessagesRejected)
		prometheus.MustRegister(a.MessagesError)
		prometheus.MustRegister(a.Incoming)
		prometheus.MustRegister(a.Outgoing)
		prometheus.MustRegister(a.RequestRejected)

	})
	a.MessagesError.Add(0)
	for _, r := range apnConfig.APN {
		a.MessagesReceived.With(prometheus.Labels{"apn": r.APN.Name}).Add(0)
		a.MessagesSent.With(prometheus.Labels{"apn": r.APN.Name}).Add(0)
		a.MessageSendErrors.With(prometheus.Labels{"apn": r.APN.Name}).Add(0)
		a.MessagesForwarded.With(prometheus.Labels{"apn": r.APN.Name}).Add(0)
		a.MessagesRejected.With(prometheus.Labels{"apn": r.APN.Name}).Add(0)
		for _, nas := range r.Ranges {
			a.Incoming.With(prometheus.Labels{
				"apn":       r.APN.Name,
				"nas":       nas.Identifier,
				"transport": "coap-push",
			}).Add(0)
			a.Incoming.With(prometheus.Labels{
				"apn":       r.APN.Name,
				"nas":       nas.Identifier,
				"transport": "coap-pull",
			}).Add(0)
			a.Incoming.With(prometheus.Labels{
				"apn":       r.APN.Name,
				"nas":       nas.Identifier,
				"transport": "udp-pull",
			}).Add(0)
			a.Incoming.With(prometheus.Labels{
				"apn":       r.APN.Name,
				"nas":       nas.Identifier,
				"transport": "udp",
			}).Add(0)
		}
	}

}

// MessageReceived increments the message received counter
func (a *APNCounters) MessageReceived(r model.NASRanges) {
	a.MessagesReceived.With(prometheus.Labels{"apn": r.APN.Name}).Inc()
}

// MessageSent incmrements the message sent counter
func (a *APNCounters) MessageSent(r model.NASRanges) {
	a.MessagesSent.With(prometheus.Labels{"apn": r.APN.Name}).Inc()
}

// MessageSendError increases the send error counter
func (a *APNCounters) MessageSendError(r model.NASRanges) {
	a.MessageSendErrors.With(prometheus.Labels{"apn": r.APN.Name}).Inc()
}

// MessageForwarded increases the forwarded counter
func (a *APNCounters) MessageForwarded(r model.NASRanges) {
	a.MessagesForwarded.With(prometheus.Labels{"apn": r.APN.Name}).Inc()
}

// MessageRejected increases the rejected counter
func (a *APNCounters) MessageRejected(r model.NASRanges) {
	a.MessagesRejected.With(prometheus.Labels{"apn": r.APN.Name}).Inc()
}

// MessageError increases the error counter
func (a *APNCounters) MessageError() {
	a.MessagesError.Inc()
}

// In increments the incoming counter
func (a *APNCounters) In(apn model.APN, nas model.NAS, t model.MessageTransport) {
	a.Incoming.With(prometheus.Labels{
		"apn":       apn.Name,
		"nas":       nas.Identifier,
		"transport": t.String(),
	}).Inc()
}

// Out increments the outgoing counter
func (a *APNCounters) Out(apn model.APN, nas model.NAS, t model.MessageTransport) {
	a.Incoming.With(prometheus.Labels{
		"apn":       apn.Name,
		"nas":       nas.Identifier,
		"transport": t.String(),
	}).Inc()
}

// Rejected increments the rejected counter
func (a *APNCounters) Rejected() {
	a.RequestRejected.Inc()
}
