package storage

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/eesrc/horde/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestAPNCache(t *testing.T) {
	assert := require.New(t)

	cache, err := NewAPNCache(&dummyStore{})
	assert.NoError(err)
	assert.NotNil(cache)

	cache.APN = make([]model.NASRanges, 0)
	assert.NoError(cache.Reload(&dummyStore{}))

	nas, ok := cache.FindNAS("N0_3")
	assert.True(ok)
	assert.Equal(30, nas.ID)

	nas, ok = cache.FindByID(21)
	assert.True(ok)
	assert.Equal(21, nas.ID)
	assert.Equal(2, nas.ApnID)
}

type dummyStore struct {
}

func (d *dummyStore) ListAPN() ([]model.APN, error) {
	return []model.APN{
		model.APN{ID: 0, Name: "mda.1"},
		model.APN{ID: 1, Name: "mda.2"},
		model.APN{ID: 2, Name: "mda.3"},
		model.APN{ID: 3, Name: "mda.4"},
	}, nil
}
func (d *dummyStore) ListNAS(apnID int) ([]model.NAS, error) {
	return []model.NAS{
		model.NAS{ID: 10 * apnID, Identifier: fmt.Sprintf("N0_%d", apnID), CIDR: "127.0.0.1/8", ApnID: apnID},
		model.NAS{ID: 10*apnID + 1, Identifier: fmt.Sprintf("N1_%d", apnID), CIDR: "127.0.0.1/8", ApnID: apnID},
	}, nil
}

// Ignored methods below
func (d *dummyStore) CreateAPN(model.APN) error {
	return errors.New("not imlpemented")
}
func (d *dummyStore) RemoveAPN(apnID int) error {
	return errors.New("not imlpemented")

}
func (d *dummyStore) CreateNAS(model.NAS) error {
	return errors.New("not imlpemented")

}
func (d *dummyStore) RemoveNAS(apnID, nasID int) error {
	return errors.New("not imlpemented")

}
func (d *dummyStore) ListAllocations(apnID, nasID, maxRows int) ([]model.Allocation, error) {
	return nil, errors.New("not imlpemented")

}
func (d *dummyStore) CreateAllocation(model.Allocation) error {
	return errors.New("not imlpemented")

}
func (d *dummyStore) RemoveAllocation(apnID int, nasID int, imsi int64) error {
	return errors.New("not imlpemented")

}
func (d *dummyStore) RetrieveAllAllocations(imsi int64) ([]model.Allocation, error) {
	return nil, errors.New("not imlpemented")

}
func (d *dummyStore) RetrieveAllocation(imsi int64, apnid int, nasid int) (model.Allocation, error) {
	return model.Allocation{}, errors.New("not imlpemented")

}
func (d *dummyStore) LookupIMSIFromIP(ip net.IP, ranges model.NASRanges) (int64, error) {
	return 0, errors.New("not imlpemented")

}
func (d *dummyStore) RetrieveNAS(apnID int, nasid int) (model.NAS, error) {
	return model.NAS{}, errors.New("not imlpemented")
}
