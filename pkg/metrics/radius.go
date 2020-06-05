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

	"github.com/eesrc/horde/pkg/storage"
	"github.com/prometheus/client_golang/prometheus"
)

// RADIUSCounters holds monitoring counters for a RADIUS endpoint
type RADIUSCounters struct {
	Accept *prometheus.CounterVec
	Reject *prometheus.CounterVec
	Alloc  *prometheus.GaugeVec
	Free   *prometheus.GaugeVec
	Reused *prometheus.CounterVec
}

var radiusInitCounters sync.Once

// NewRADIUSCounters creates a new set of counters for a RADIUS endpoint.
// TODO(stalehd): Add an "apn" dimension to these metrics later.
func NewRADIUSCounters() *RADIUSCounters {
	return &RADIUSCounters{
		Accept: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "apn_radius_accept",
			Help: "Accept responses from RADIUS server",
		}, []string{"nas"}),
		Reject: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "apn_radius_reject",
			Help: "Reject responses from RADIUS server",
		}, []string{"nas"}),
		Alloc: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "apn_radius_allocated_ip",
			Help: "Number of allocated IP addresses",
		}, []string{"nas"}),
		Free: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "apn_radius_free_ip",
			Help: "Number of available IPs in pool",
		}, []string{"nas"}),
		Reused: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "apn_radius_reuse_ip",
			Help: "Number of reused IPs",
		}, []string{"nas"}),
	}
}

// Start registers the RADIUS counters
func (r *RADIUSCounters) Start(apnConfig *storage.APNConfigCache) {
	radiusInitCounters.Do(func() {
		prometheus.MustRegister(r.Accept)
		prometheus.MustRegister(r.Reject)
		prometheus.MustRegister(r.Alloc)
		prometheus.MustRegister(r.Free)
		prometheus.MustRegister(r.Reused)
	})
	for _, apn := range apnConfig.APN {
		for _, nas := range apn.Ranges {
			r.Accept.With(prometheus.Labels{"nas": nas.Identifier}).Add(0)
			r.Reject.With(prometheus.Labels{"nas": nas.Identifier}).Add(0)
			r.Alloc.With(prometheus.Labels{"nas": nas.Identifier}).Add(0)
			r.Free.With(prometheus.Labels{"nas": nas.Identifier}).Add(0)
			r.Reused.With(prometheus.Labels{"nas": nas.Identifier}).Add(0)
		}
	}
}

// AcceptRequest increments the Accept counter
func (r *RADIUSCounters) AcceptRequest(nasIdentifier string) {
	r.Accept.With(prometheus.Labels{"nas": nasIdentifier}).Inc()
}

// RejectRequest increments the Reject counter
func (r *RADIUSCounters) RejectRequest(nasIdentifier string) {
	r.Reject.With(prometheus.Labels{"nas": nasIdentifier}).Inc()
}

// IPAllocated increments the  Alloc counter and decrements the Free counter
func (r *RADIUSCounters) IPAllocated(nasIdentifier string) {
	r.Alloc.With(prometheus.Labels{"nas": nasIdentifier}).Inc()
	r.Free.With(prometheus.Labels{"nas": nasIdentifier}).Dec()
}

// IPReleased decrements the Alloc counter and increments the Free counter
func (r *RADIUSCounters) IPReleased(nasIdentifier string) {
	r.Alloc.With(prometheus.Labels{"nas": nasIdentifier}).Dec()
	r.Free.With(prometheus.Labels{"nas": nasIdentifier}).Inc()
}

// IPReused decrements the Alloc counter and increments the Free counter
func (r *RADIUSCounters) IPReused(nasIdentifier string) {
	r.Reused.With(prometheus.Labels{"nas": nasIdentifier}).Inc()
}

// UpdateIPAllocation sets the allocation counts
func (r *RADIUSCounters) UpdateIPAllocation(nasIdentifier string, allocated, free int) {
	r.Alloc.With(prometheus.Labels{"nas": nasIdentifier}).Set(float64(allocated))
	r.Free.With(prometheus.Labels{"nas": nasIdentifier}).Set(float64(free))
}
