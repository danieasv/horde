package storage

import (
	"sync"

	"github.com/ExploratoryEngineering/logging"
	"github.com/eesrc/horde/pkg/model"
)

// APNConfigCache holds a cached APN configuration. This is used mainly in the
// IP allocation code. The in-memory caching can replace this but this method
// is much quicker wrt lookups since there's many orders of magnitude less
// data to sift through.
type APNConfigCache struct {
	APN   []model.NASRanges
	mutex *sync.Mutex
}

// NewAPNCache creates and populates a new APN cache
func NewAPNCache(apnStore APNStore) (*APNConfigCache, error) {
	ret := &APNConfigCache{
		APN:   make([]model.NASRanges, 0),
		mutex: &sync.Mutex{}}
	return ret, ret.Reload(apnStore)

}

// FindNAS finds the matching NAS
func (a *APNConfigCache) FindNAS(identifier string) (model.NAS, bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	for _, apn := range a.APN {
		for _, nas := range apn.Ranges {
			if nas.Identifier == identifier {
				return nas, true
			}
		}
	}
	return model.NAS{}, false
}

// FindByID locates the NAS with the matching ID
func (a *APNConfigCache) FindByID(nasID int) (model.NAS, bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	for _, apn := range a.APN {
		for _, nas := range apn.Ranges {
			if nas.ID == nasID {
				return nas, true
			}
		}
	}
	return model.NAS{}, false
}

// FindAPN locates the APN (and NAS entries) for the APN with the matching ID
func (a *APNConfigCache) FindAPN(apnID int) (model.NASRanges, bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	for _, apn := range a.APN {
		if apn.APN.ID == apnID {
			return apn, true
		}
	}
	return model.NASRanges{}, false
}

// Reload reloads the cache. Clients will be blocked while the cache is
// updated
func (a *APNConfigCache) Reload(apnStore APNStore) error {
	apns, err := apnStore.ListAPN()
	if err != nil {
		return err
	}
	newConfig := make([]model.NASRanges, 0)
	apnCount := 0
	nasCount := 0
	var nasList []model.NAS
	for _, v := range apns {
		nases, err := apnStore.ListNAS(v.ID)
		if err != nil {
			return err
		}
		nasList = append(nasList, nases...)
		newConfig = append(newConfig, model.NASRanges{APN: v, Ranges: nasList})
		apnCount++
		nasCount += len(nasList)
	}
	a.mutex.Lock()
	a.APN = newConfig
	defer a.mutex.Unlock()

	logging.Debug("Loaded config with %d APNs and %d NASes", apnCount, nasCount)
	return nil
}
