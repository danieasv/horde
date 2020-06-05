package api

//
// Copyright 2020 Telenor Digital AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
import (
	"context"
	"errors"
	"testing"

	"github.com/ExploratoryEngineering/pubsub"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/output"
	"github.com/eesrc/horde/pkg/output/outputconfig"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/eesrc/horde/pkg/storage/sqlstore"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type outputTestSetup struct {
	assert        *require.Assertions
	store         storage.DataStore
	outputService outputService
	user          model.User
	ctx           context.Context
	collection    model.Collection
	output        model.Output
	mgr           output.Manager
}

func newOutputTest(t *testing.T) outputTestSetup {
	ret := outputTestSetup{}
	ret.assert = require.New(t)
	ret.store = sqlstore.NewMemoryStore()
	ret.mgr = newDummyManager()
	ret.outputService = newOutputService(ret.store, ret.mgr, model.FieldMaskParameters{})
	ret.assert.NotNil(ret.outputService)

	ret.user, _, ret.ctx = createAuthenticatedContext(ret.assert, ret.store)

	ret.collection = model.NewCollection()
	ret.collection.ID = ret.store.NewCollectionID()
	ret.collection.TeamID = ret.user.PrivateTeamID
	ret.assert.NoError(ret.store.CreateCollection(ret.user.ID, ret.collection))

	ret.output = model.NewOutput()
	ret.output.ID = ret.store.NewOutputID()
	ret.output.CollectionID = ret.collection.ID
	ret.output.Enabled = false
	ret.output.Type = "webhook"
	ret.output.Config = model.NewOutputConfig()
	output.DisableLocalhostChecks()
	ret.output.Config[outputconfig.WebhookURLField] = "http://127.0.0.1"
	ret.assert.NoError(ret.store.CreateOutput(ret.user.ID, ret.output))
	ret.assert.NoError(ret.mgr.Update(ret.output, 0))
	return ret
}
func TestOutputTags(t *testing.T) {
	ot := newOutputTest(t)
	doTagTests(ot.ctx, ot.assert, ot.collection.ID.String(), true, ot.output.ID.String(), tagFunctions{
		ListTags:   ot.outputService.ListOutputTags,
		UpdateTag:  ot.outputService.UpdateOutputTag,
		GetTag:     ot.outputService.GetOutputTag,
		DeleteTag:  ot.outputService.DeleteOutputTag,
		UpdateTags: ot.outputService.UpdateOutputTags,
	})
}

// Factory type for *apipb.OutputRequest
type outputRequestFactory struct {
}

func (orf *outputRequestFactory) ValidRequest() interface{} {
	return &apipb.OutputRequest{}
}

func (orf *outputRequestFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.OutputRequest).CollectionId = cid
}

func (orf *outputRequestFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	req.(*apipb.OutputRequest).OutputId = oid
}

func TestRetrieveOutput(t *testing.T) {
	ot := newOutputTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      ot.ctx,
		Assert:                    ot.assert,
		CollectionID:              ot.collection.ID.String(),
		IdentifierID:              ot.output.ID.String(),
		TestWithInvalidIdentifier: true,
		RequestFactory:            &outputRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ot.outputService.RetrieveOutput(ctx, nil)
			}
			return ot.outputService.RetrieveOutput(ctx, req.(*apipb.OutputRequest))
		}})
}

func TestDeleteOutput(t *testing.T) {
	ot := newOutputTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      ot.ctx,
		Assert:                    ot.assert,
		CollectionID:              ot.collection.ID.String(),
		IdentifierID:              ot.output.ID.String(),
		TestWithInvalidIdentifier: true,
		RequestFactory:            &outputRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ot.outputService.DeleteOutput(ctx, nil)
			}
			return ot.outputService.DeleteOutput(ctx, req.(*apipb.OutputRequest))
		}})

	// A 2nd delete should return not found
	req := &apipb.OutputRequest{
		CollectionId: &wrappers.StringValue{Value: ot.collection.ID.String()},
		OutputId:     &wrappers.StringValue{Value: ot.output.ID.String()},
	}
	_, err := ot.outputService.DeleteOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Create a new output where the user is a non-admin
	u2, _, _ := createAuthenticatedContext(ot.assert, ot.store)
	t2 := model.NewTeam()
	t2.AddMember(model.NewMember(ot.user, model.MemberRole))
	t2.AddMember(model.NewMember(u2, model.AdminRole))
	ot.assert.NoError(ot.store.CreateTeam(t2))

	c2 := model.NewCollection()
	c2.TeamID = t2.ID
	c2.ID = ot.store.NewCollectionID()
	ot.assert.NoError(ot.store.CreateCollection(u2.ID, c2))

	o2 := model.NewOutput()
	o2.ID = ot.store.NewOutputID()
	o2.CollectionID = c2.ID
	o2.Enabled = true
	o2.Config = model.NewOutputConfig()
	ot.assert.NoError(ot.store.CreateOutput(u2.ID, o2))

	req.CollectionId = &wrappers.StringValue{Value: c2.ID.String()}
	req.OutputId = &wrappers.StringValue{Value: o2.ID.String()}

	_, err = ot.outputService.DeleteOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())
}

func TestOutputLogs(t *testing.T) {
	ot := newOutputTest(t)

	genericRequestTests(tparam{
		AuthenticatedContext:      ot.ctx,
		Assert:                    ot.assert,
		CollectionID:              ot.collection.ID.String(),
		IdentifierID:              ot.output.ID.String(),
		TestWithInvalidIdentifier: true,
		RequestFactory:            &outputRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ot.outputService.Logs(ctx, nil)
			}
			return ot.outputService.Logs(ctx, req.(*apipb.OutputRequest))
		}})

	// Stop the output - should return FailedPrecondition
	ot.mgr.Stop(ot.output.ID)
	req := &apipb.OutputRequest{
		CollectionId: &wrappers.StringValue{Value: ot.collection.ID.String()},
		OutputId:     &wrappers.StringValue{Value: ot.output.ID.String()},
	}
	_, err := ot.outputService.Logs(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.FailedPrecondition.String(), status.Code(err).String())
}

func TestOutputStatus(t *testing.T) {
	ot := newOutputTest(t)
	genericRequestTests(tparam{
		AuthenticatedContext:      ot.ctx,
		Assert:                    ot.assert,
		CollectionID:              ot.collection.ID.String(),
		IdentifierID:              ot.output.ID.String(),
		TestWithInvalidIdentifier: true,
		RequestFactory:            &outputRequestFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ot.outputService.Status(ctx, nil)
			}
			return ot.outputService.Status(ctx, req.(*apipb.OutputRequest))
		}})
	ot.mgr.Stop(ot.output.ID)
	req := &apipb.OutputRequest{
		CollectionId: &wrappers.StringValue{Value: ot.collection.ID.String()},
		OutputId:     &wrappers.StringValue{Value: ot.output.ID.String()},
	}
	_, err := ot.outputService.Status(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.FailedPrecondition.String(), status.Code(err).String())
}

type outputFactory struct {
}

func (of *outputFactory) ValidRequest() interface{} {
	return &apipb.Output{
		Type: apipb.Output_webhook,
		Config: &apipb.OutputConfig{
			Url: &wrappers.StringValue{Value: "http://127.0.0.1/"},
		},
		Tags: map[string]string{
			"Name": "The output",
		},
		Enabled: &wrappers.BoolValue{Value: true},
	}
}

func (of *outputFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.Output).CollectionId = cid
}

func (of *outputFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	req.(*apipb.Output).OutputId = oid
}

func TestCreateOutput(t *testing.T) {
	ot := newOutputTest(t)
	genericRequestTests(tparam{
		AuthenticatedContext:      ot.ctx,
		Assert:                    ot.assert,
		CollectionID:              ot.collection.ID.String(),
		IdentifierID:              "",
		TestWithInvalidIdentifier: false,
		RequestFactory:            &outputFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ot.outputService.CreateOutput(ctx, nil)
			}
			return ot.outputService.CreateOutput(ctx, req.(*apipb.Output))
		}})

	// Nil config => error
	req := &apipb.Output{
		CollectionId: &wrappers.StringValue{Value: ot.collection.ID.String()},
		Type:         apipb.Output_webhook,
		Config:       nil,
	}
	_, err := ot.outputService.CreateOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Invalid tags => error
	req.Config = &apipb.OutputConfig{
		Url: &wrappers.StringValue{Value: "http://127.0.0.1"},
	}
	req.Tags = map[string]string{
		"": "Invalid value",
	}
	_, err = ot.outputService.CreateOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Invalid config => error
	req = &apipb.Output{
		CollectionId: &wrappers.StringValue{Value: ot.collection.ID.String()},
		Type:         apipb.Output_webhook,
		Config: &apipb.OutputConfig{
			BasicAuthPass: &wrappers.StringValue{Value: "thiswontwork"},
		},
		Enabled: &wrappers.BoolValue{Value: false},
	}
	_, err = ot.outputService.CreateOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Unknown collection - NotFound error
	req = &apipb.Output{
		CollectionId: &wrappers.StringValue{Value: "0"},
		Type:         apipb.Output_webhook,
		Config: &apipb.OutputConfig{
			Url: &wrappers.StringValue{Value: "http://127.0.0.1"},
		},
		Enabled: &wrappers.BoolValue{Value: false},
	}
	_, err = ot.outputService.CreateOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Collection owned by someone else => error
	u2, _, _ := createAuthenticatedContext(ot.assert, ot.store)
	t2 := model.NewTeam()
	t2.AddMember(model.NewMember(ot.user, model.MemberRole))
	t2.AddMember(model.NewMember(u2, model.AdminRole))
	ot.assert.NoError(ot.store.CreateTeam(t2))

	c2 := model.NewCollection()
	c2.TeamID = t2.ID
	c2.ID = ot.store.NewCollectionID()
	ot.assert.NoError(ot.store.CreateCollection(u2.ID, c2))

	o2 := model.NewOutput()
	o2.ID = ot.store.NewOutputID()
	o2.CollectionID = c2.ID
	o2.Enabled = true
	o2.Config = model.NewOutputConfig()
	ot.assert.NoError(ot.store.CreateOutput(u2.ID, o2))

	req.CollectionId = &wrappers.StringValue{Value: c2.ID.String()}
	req.OutputId = &wrappers.StringValue{Value: o2.ID.String()}
	_, err = ot.outputService.CreateOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())

}

// Factory type for *apipb.OutputRequest
type listOutputFactory struct {
}

func (lof *listOutputFactory) ValidRequest() interface{} {
	return &apipb.ListOutputRequest{}
}

func (lof *listOutputFactory) SetCollection(req interface{}, cid *wrappers.StringValue) {
	req.(*apipb.ListOutputRequest).CollectionId = cid
}

func (lof *listOutputFactory) SetIdentifier(req interface{}, oid *wrappers.StringValue) {
	// Nothing
}

func TestListOutput(t *testing.T) {
	ot := newOutputTest(t)
	genericRequestTests(tparam{
		AuthenticatedContext:      ot.ctx,
		Assert:                    ot.assert,
		CollectionID:              ot.collection.ID.String(),
		IdentifierID:              "",
		TestWithInvalidIdentifier: false,
		RequestFactory:            &listOutputFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ot.outputService.ListOutputs(ctx, nil)
			}
			return ot.outputService.ListOutputs(ctx, req.(*apipb.ListOutputRequest))
		}})
}

func TestUpdateOutput(t *testing.T) {
	ot := newOutputTest(t)
	genericRequestTests(tparam{
		AuthenticatedContext:      ot.ctx,
		Assert:                    ot.assert,
		CollectionID:              ot.collection.ID.String(),
		IdentifierID:              ot.output.ID.String(),
		TestWithInvalidIdentifier: true,
		RequestFactory:            &outputFactory{},
		RequestFunc: func(ctx context.Context, req interface{}) (interface{}, error) {
			if req == nil {
				return ot.outputService.UpdateOutput(ctx, nil)
			}
			return ot.outputService.UpdateOutput(ctx, req.(*apipb.Output))
		}})

	// Change enabled, type and tags in one go
	req := &apipb.Output{
		CollectionId: &wrappers.StringValue{Value: ot.collection.ID.String()},
		OutputId:     &wrappers.StringValue{Value: ot.output.ID.String()},
		Enabled:      &wrappers.BoolValue{Value: false},
		Tags: map[string]string{
			"Name":      "some name",
			"Other tag": "Some other tag",
		},
		Type: apipb.Output_udp,
		Config: &apipb.OutputConfig{
			Host: &wrappers.StringValue{Value: "127.0.0.1"},
			Port: &wrappers.Int32Value{Value: 8080},
		},
	}
	res, err := ot.outputService.UpdateOutput(ot.ctx, req)
	ot.assert.NoError(err)
	ot.assert.NotNil(res)

	// Invalid config => error
	req = &apipb.Output{
		CollectionId: &wrappers.StringValue{Value: ot.collection.ID.String()},
		OutputId:     &wrappers.StringValue{Value: ot.output.ID.String()},
		Type:         apipb.Output_webhook,
		Config:       &apipb.OutputConfig{},
	}
	_, err = ot.outputService.UpdateOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Just toggle enabled/disabled
	req = &apipb.Output{
		CollectionId: &wrappers.StringValue{Value: ot.collection.ID.String()},
		OutputId:     &wrappers.StringValue{Value: ot.output.ID.String()},
		Enabled:      &wrappers.BoolValue{Value: true},
	}
	res, err = ot.outputService.UpdateOutput(ot.ctx, req)
	ot.assert.NoError(err)
	ot.assert.NotNil(res)

	// Invalid tags => error
	req = &apipb.Output{
		CollectionId: &wrappers.StringValue{Value: ot.collection.ID.String()},
		OutputId:     &wrappers.StringValue{Value: ot.output.ID.String()},
		Tags: map[string]string{
			"": "Invalid tag",
		},
	}
	_, err = ot.outputService.UpdateOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Unknown collection ID => error
	req = &apipb.Output{
		CollectionId: &wrappers.StringValue{Value: "0"},
		OutputId:     &wrappers.StringValue{Value: ot.output.ID.String()},
		Tags: map[string]string{
			"Name": "Updated",
		},
	}
	_, err = ot.outputService.UpdateOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Collection owned by someone else => error
	u2, _, _ := createAuthenticatedContext(ot.assert, ot.store)
	t2 := model.NewTeam()
	t2.AddMember(model.NewMember(ot.user, model.MemberRole))
	t2.AddMember(model.NewMember(u2, model.AdminRole))
	ot.assert.NoError(ot.store.CreateTeam(t2))

	c2 := model.NewCollection()
	c2.TeamID = t2.ID
	c2.ID = ot.store.NewCollectionID()
	ot.assert.NoError(ot.store.CreateCollection(u2.ID, c2))

	o2 := model.NewOutput()
	o2.ID = ot.store.NewOutputID()
	o2.CollectionID = c2.ID
	o2.Enabled = true
	o2.Type = "webhook"
	o2.Config = model.NewOutputConfig()
	output.DisableLocalhostChecks()
	o2.Config[outputconfig.WebhookURLField] = "http://127.0.0.1/"
	ot.assert.NoError(ot.store.CreateOutput(u2.ID, o2))
	req = &apipb.Output{
		CollectionId: &wrappers.StringValue{Value: c2.ID.String()},
		OutputId:     &wrappers.StringValue{Value: o2.ID.String()},
		Tags: map[string]string{
			"Name": "Updated",
		},
	}
	_, err = ot.outputService.UpdateOutput(ot.ctx, req)
	ot.assert.Error(err)
	ot.assert.Equal(codes.PermissionDenied.String(), status.Code(err).String())
}

// ----------------------------------------------------------------------------
// Dummy output manager that will accept all outputs and pretend they are
// running just fine

type dummyManager struct {
	router  pubsub.EventRouter
	outputs map[model.OutputKey]output.Output
}

func (m *dummyManager) Refresh(ops []model.Output, fm model.FieldMask) {
	for _, v := range ops {
		m.Update(v, fm)
	}
}

func (m *dummyManager) Update(o model.Output, fm model.FieldMask) error {
	op, err := output.NewOutput(o.Type)
	if err != nil {
		return err
	}
	m.outputs[o.ID] = op
	return nil
}

func (m *dummyManager) Stop(id model.OutputKey) error {
	delete(m.outputs, id)
	return nil
}

func (m *dummyManager) Shutdown() {
	// Nothing
}

func (m *dummyManager) Get(id model.OutputKey) (output.Output, error) {
	ret, ok := m.outputs[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return ret, nil
}

func (m *dummyManager) Verify(config model.Output) (model.ErrorMessage, error) {
	op, err := output.NewOutput(config.Type)
	if err != nil {
		return nil, err
	}
	return op.Validate(config.Config)
}

func (m *dummyManager) Publish(msg model.DataMessage) {
	m.router.Publish(msg.Device.CollectionID, msg)
}

func (m *dummyManager) Subscribe(collectionID model.CollectionKey) <-chan interface{} {
	return m.router.Subscribe(collectionID)
}

func (m *dummyManager) Unsubscribe(ch <-chan interface{}) {
	m.router.Unsubscribe(ch)
}

// NewDummyManager For testing: Return a dummy manager
func newDummyManager() output.Manager {
	return &dummyManager{
		router:  pubsub.NewEventRouter(2),
		outputs: make(map[model.OutputKey]output.Output),
	}
}
