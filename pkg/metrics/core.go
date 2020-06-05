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
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// CoreCounters contains counters for the horde core API; users, devices, teams et al.
type CoreCounters struct {
	UserCount              prometheus.Gauge       // Total users counter
	CollectionCount        prometheus.Gauge       // Total collections
	DeviceCount            prometheus.Gauge       // Total devices
	TeamCount              prometheus.Gauge       // Total teams
	OutputCount            prometheus.Gauge       // Total outputs
	AuthConnectCount       prometheus.Counter     // Authenticated requests via CONNECT ID, incl cookies
	AuthGithubCount        prometheus.Counter     // Authenticated requests via GitHub, incl cookies
	AuthTokenCount         prometheus.Counter     // Authenticated requests with tokens
	MessagesInCount        prometheus.Counter     // Incoming messages (aka upstream)
	MessagesOutCount       prometheus.Counter     // Sent messages (aka downstream)
	MessagesForwardMQTT    prometheus.Counter     // Messages forwarded to MQTT outputs
	MessagesForwardUDP     prometheus.Counter     // Messages forwarded to UDP outputs
	MessagesForwardWebhook prometheus.Counter     // Messages forwarded to webhooks
	MessagesForwardIFTTT   prometheus.Counter     // Messages forwarded to webhooks
	HTTPResponse           *prometheus.CounterVec // Responses from HTTP API
	InvitesCreated         prometheus.Counter
	InvitesAccepted        prometheus.Counter
	HTTPResponseTime       *prometheus.HistogramVec
}

// CounterStore is a type that reports the initial values for the performance counters.
// TODO(stalehd): Consider removing this.  Just added complexity
type CounterStore interface {
	Users() int64
	Collections() int64
	Devices() int64
	Teams() int64
	Outputs() int64
}

// NewCoreCounters creates a new set of Horde core counters with a prefix
func NewCoreCounters() *CoreCounters {

	ret := &CoreCounters{
		UserCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "user_count",
			Help: "Number of registered users",
		}),
		CollectionCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "collection_count",
			Help: "Number of collections",
		}),
		DeviceCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "device_count",
			Help: "Number of devices",
		}),
		TeamCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "team_count",
			Help: "Number of teams",
		}),
		OutputCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "output_count",
			Help: "Number of outputs",
		}),
		AuthConnectCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "auth_connect",
			Help: "CONNECT ID authentication",
		}),
		AuthGithubCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "auth_github",
			Help: "GitHub OAuth authentication",
		}),
		AuthTokenCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "auth_token",
			Help: "API token authentication",
		}),
		MessagesInCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "messages_in",
			Help: "Number of incoming messages from APN",
		}),
		MessagesOutCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "messages_out",
			Help: "Number of outgoing messages to APN",
		}),
		MessagesForwardMQTT: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "forward_mqtt",
			Help: "Messages forwarded to MQTT brokers",
		}),
		MessagesForwardUDP: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "forward_udp",
			Help: "Messages forwarded via UDP",
		}),
		MessagesForwardWebhook: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "forward_webhook",
			Help: "Messages forwarded to webhooks",
		}),
		MessagesForwardIFTTT: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "forward_ifttt",
			Help: "Messages forwarded to IFTTT",
		}),
		HTTPResponse: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_responses",
			Help: "HTTP status codes served to clients",
		}, []string{"status"}),
		InvitesAccepted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "invites_accepted",
			Help: "Invites accepted",
		}),
		InvitesCreated: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "invites_created",
			Help: "Invites created",
		}),
		HTTPResponseTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_response_time",
			Help:    "Response time (in ms) for a HTTP request",
			Buckets: []float64{1, 2, 3, 4, 5, 10, 50, 100, 250, 500, 1000},
		}, []string{"method"}),
	}

	return ret
}

var coreInitCounters sync.Once

// Start registers the counters
func (c *CoreCounters) Start() {
	coreInitCounters.Do(func() {
		prometheus.MustRegister(c.UserCount)
		prometheus.MustRegister(c.CollectionCount)
		prometheus.MustRegister(c.DeviceCount)
		prometheus.MustRegister(c.TeamCount)
		prometheus.MustRegister(c.OutputCount)
		prometheus.MustRegister(c.AuthConnectCount)
		prometheus.MustRegister(c.AuthGithubCount)
		prometheus.MustRegister(c.AuthTokenCount)
		prometheus.MustRegister(c.MessagesInCount)
		prometheus.MustRegister(c.MessagesOutCount)
		prometheus.MustRegister(c.MessagesForwardMQTT)
		prometheus.MustRegister(c.MessagesForwardUDP)
		prometheus.MustRegister(c.MessagesForwardWebhook)
		prometheus.MustRegister(c.MessagesForwardIFTTT)
		prometheus.MustRegister(c.HTTPResponse)
		prometheus.MustRegister(c.InvitesCreated)
		prometheus.MustRegister(c.InvitesAccepted)
		prometheus.MustRegister(c.HTTPResponseTime)
	})
	c.TeamCount.Set(0)
	c.UserCount.Set(0)
	c.CollectionCount.Set(0)
	c.DeviceCount.Set(0)
	c.OutputCount.Set(0)
	c.AuthConnectCount.Add(0)
	c.AuthGithubCount.Add(0)
	c.AuthTokenCount.Add(0)
	c.MessagesInCount.Add(0)
	c.MessagesOutCount.Add(0)
	c.MessagesForwardMQTT.Add(0)
	c.MessagesForwardUDP.Add(0)
	c.MessagesForwardWebhook.Add(0)
	c.MessagesForwardIFTTT.Add(0)
	c.HTTPResponse.With(prometheus.Labels{"status": "200"}).Add(0)
	c.HTTPResponse.With(prometheus.Labels{"status": "201"}).Add(0)
	c.HTTPResponse.With(prometheus.Labels{"status": "204"}).Add(0)
	c.HTTPResponse.With(prometheus.Labels{"status": "400"}).Add(0)
	c.HTTPResponse.With(prometheus.Labels{"status": "401"}).Add(0)
	c.HTTPResponse.With(prometheus.Labels{"status": "404"}).Add(0)
	c.HTTPResponse.With(prometheus.Labels{"status": "409"}).Add(0)
	c.HTTPResponse.With(prometheus.Labels{"status": "500"}).Add(0)

	c.InvitesCreated.Add(0)
	c.InvitesAccepted.Add(0)

	c.HTTPResponseTime.With(prometheus.Labels{"method": "GET"}).Observe(0)
	c.HTTPResponseTime.With(prometheus.Labels{"method": "POST"}).Observe(0)
	c.HTTPResponseTime.With(prometheus.Labels{"method": "PATCH"}).Observe(0)
	c.HTTPResponseTime.With(prometheus.Labels{"method": "DELETE"}).Observe(0)
	c.HTTPResponseTime.With(prometheus.Labels{"method": "OPTIONS"}).Observe(0)
	c.HTTPResponseTime.With(prometheus.Labels{"method": "HEAD"}).Observe(0)
}

// Update updates the counters with values from the counter store
func (c *CoreCounters) Update(counterStore CounterStore) {
	c.UserCount.Set(float64(counterStore.Users()))
	c.CollectionCount.Set(float64(counterStore.Collections()))
	c.DeviceCount.Set(float64(counterStore.Devices()))
	c.TeamCount.Set(float64(counterStore.Teams()))
	c.OutputCount.Set(float64(counterStore.Outputs()))
}

// AddHTTPStatus increments the counters for the HTTP statuses
func (c *CoreCounters) AddHTTPStatus(status int) {
	c.HTTPResponse.With(prometheus.Labels{"status": fmt.Sprintf("%d", status)}).Inc()
}

// AddHTTPResponseTime registers response time for a HTTP handler
func (c *CoreCounters) AddHTTPResponseTime(method string, t time.Duration) {
	c.HTTPResponseTime.With(prometheus.Labels{"method": method}).Observe(float64(t / time.Millisecond))
}
