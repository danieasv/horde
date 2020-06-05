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
	"time"

	"github.com/eesrc/horde/pkg/addons/magpie/datastore"
	"github.com/eesrc/horde/pkg/api/apipb"
	"github.com/eesrc/horde/pkg/ghlogin"
	"github.com/eesrc/horde/pkg/model"
	"github.com/eesrc/horde/pkg/storage"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Common service test functions

// Helper function to create a user and a token..
func createUserAndToken(assert *require.Assertions, authType model.AuthMethod, store storage.DataStore) (model.User, string) {
	// Create user with a single token
	team := model.NewTeam()
	team.ID = store.NewTeamID()
	userID := store.NewUserID()
	user := model.NewUser(userID, userID.String(), authType, team.ID)
	assert.NoError(store.CreateUser(user, team))
	token := model.NewToken()
	token.UserID = user.ID
	token.Resource = "/"
	token.Write = true
	assert.NoError(token.GenerateToken())
	assert.NoError(store.CreateToken(token))
	return user, token.Token
}

// Create user, token and context
func createAuthenticatedContext(assert *require.Assertions, store storage.DataStore) (model.User, string, context.Context) {
	user, token := createUserAndToken(assert, model.AuthGitHub, store)
	dummyGithubSession := ghlogin.Profile{Login: user.ExternalID}
	ctx := context.WithValue(context.Background(), ghlogin.GitHubSessionProfile, dummyGithubSession)
	return user, token, ctx
}

type tagFunctions struct {
	ListTags   func(context.Context, *apipb.TagRequest) (*apipb.TagResponse, error)
	UpdateTags func(ctx context.Context, req *apipb.UpdateTagRequest) (*apipb.TagResponse, error)
	GetTag     func(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error)
	DeleteTag  func(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error)
	UpdateTag  func(ctx context.Context, req *apipb.TagRequest) (*apipb.TagValueResponse, error)
}

func doTagTests(authContext context.Context, assert *require.Assertions, collectionID string, withCollection bool, knownID string, tagService tagFunctions) {
	doListTagsTest(authContext, assert, collectionID, withCollection, knownID, tagService)
	doUpdateTagsTest(authContext, assert, collectionID, withCollection, knownID, tagService)
	doGetDeleteUpdateTag(authContext, assert, collectionID, withCollection, knownID, tagService)
}

func doListTagsTest(authContext context.Context, assert *require.Assertions, collectionID string, withCollection bool, knownID string, tagService tagFunctions) {
	// Test nil request objects and parameters
	_, err := tagService.ListTags(authContext, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = tagService.ListTags(authContext, &apipb.TagRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	if withCollection {
		_, err = tagService.ListTags(authContext, &apipb.TagRequest{
			Identifier: &wrappers.StringValue{Value: "0"},
		})
		assert.Error(err)
		assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	}
	// Test unauthenticated request
	r := &apipb.TagRequest{
		Identifier: &wrappers.StringValue{Value: knownID},
	}
	if withCollection {
		r.CollectionId = &wrappers.StringValue{Value: collectionID}
	}
	_, err = tagService.ListTags(context.Background(), r)
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Test with unknown ID. The ID is (usually) numeric so we'll use just "0"
	if withCollection {
		r.CollectionId = &wrappers.StringValue{Value: "0"}
		_, err = tagService.ListTags(authContext, r)
		assert.Error(err)
		assert.Equal(codes.NotFound.String(), status.Code(err).String())
	}
	r.Identifier = &wrappers.StringValue{Value: "0"}
	_, err = tagService.ListTags(authContext, r)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// List tags on object. Should succeed
	r.Identifier = &wrappers.StringValue{Value: knownID}
	if withCollection {
		r.CollectionId = &wrappers.StringValue{Value: collectionID}
	}
	res, err := tagService.ListTags(authContext, r)
	assert.NoError(err)
	assert.NotNil(res)

}

func doUpdateTagsTest(authContext context.Context, assert *require.Assertions, collectionID string, withCollection bool, knownID string, tagService tagFunctions) {
	// Nil request object and nil identifier
	_, err := tagService.UpdateTags(authContext, nil)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = tagService.UpdateTags(authContext, &apipb.UpdateTagRequest{})
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	r := &apipb.UpdateTagRequest{
		Identifier: &wrappers.StringValue{Value: knownID},
		Tags:       map[string]string{"foo": "bar"},
	}
	if withCollection {
		r.CollectionId = &wrappers.StringValue{Value: collectionID}
	}
	// Unauthenticated check
	_, err = tagService.UpdateTags(context.Background(), r)
	assert.Error(err)
	assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Unknown ID
	r.Identifier = &wrappers.StringValue{Value: "0"}
	if withCollection {
		r.CollectionId = &wrappers.StringValue{Value: "0"}
	}
	_, err = tagService.UpdateTags(authContext, r)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Missing tags field
	r.Identifier = &wrappers.StringValue{Value: knownID}
	if withCollection {
		r.CollectionId = &wrappers.StringValue{Value: collectionID}
	}
	r.Tags = nil
	_, err = tagService.UpdateTags(authContext, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Invalid tag name value
	r.Tags = map[string]string{
		"foo":    "bar",
		"":       "",
		"other2": "   ",
	}
	_, err = tagService.UpdateTags(authContext, r)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Valid token and tags
	r.Tags = map[string]string{
		"foo":    "bar",
		"other":  "",
		"other2": "   ",
	}
	tags, err := tagService.UpdateTags(authContext, r)
	assert.NoError(err)
	assert.Contains(tags.Tags, "foo")
	assert.NotContains(tags.Tags, "other")
	assert.NotContains(tags.Tags, "other2")

}

func doGetDeleteUpdateTag(authContext context.Context, assert *require.Assertions, collectionID string, withCollection bool, knownID string, tagService tagFunctions) {
	// Unauthenticated check
	req := &apipb.TagRequest{
		Identifier: &wrappers.StringValue{Value: knownID},
		Name:       &wrappers.StringValue{Value: "Bar"},
	}
	if withCollection {
		req.CollectionId = &wrappers.StringValue{Value: collectionID}
	}
	_, err := tagService.GetTag(context.Background(), req)
	assert.Error(err)
	_, err = tagService.UpdateTag(context.Background(), req)
	assert.Error(err)
	_, err = tagService.DeleteTag(context.Background(), req)
	assert.Error(err)

	// Test with nil request
	_, err = tagService.GetTag(authContext, nil)
	assert.Error(err)
	_, err = tagService.UpdateTag(authContext, nil)
	assert.Error(err)
	_, err = tagService.DeleteTag(authContext, nil)
	assert.Error(err)

	// Test with unknown identifier. Should return not found for all
	req.Identifier = &wrappers.StringValue{Value: "0"}
	if withCollection {
		req.CollectionId = &wrappers.StringValue{Value: "0"}
	}
	req.Name = &wrappers.StringValue{Value: "foo"}
	_, err = tagService.GetTag(authContext, req)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	_, err = tagService.UpdateTag(authContext, req)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	_, err = tagService.DeleteTag(authContext, req)
	assert.Error(err)
	assert.Equal(codes.NotFound.String(), status.Code(err).String())

	// Test with empty tag name
	req.Identifier = &wrappers.StringValue{Value: knownID}
	if withCollection {
		req.CollectionId = &wrappers.StringValue{Value: collectionID}
	}
	req.Name = &wrappers.StringValue{Value: "   "}
	req.Value = &wrappers.StringValue{Value: "Something"}
	_, err = tagService.GetTag(authContext, req)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = tagService.UpdateTag(authContext, req)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	_, err = tagService.DeleteTag(authContext, req)
	assert.Error(err)
	assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Update (to ensure initial value), get, update, get, delete, get in that order.
	req.Name = &wrappers.StringValue{Value: "foo"}
	req.Value = &wrappers.StringValue{Value: "bar"}

	res, err := tagService.UpdateTag(authContext, req)
	assert.NoError(err)
	assert.Equal("bar", res.Value.Value)

	res, err = tagService.GetTag(authContext, req)
	assert.NoError(err)
	assert.Equal("bar", res.Value.Value)

	req.Value = &wrappers.StringValue{Value: "baz"}
	res, err = tagService.UpdateTag(authContext, req)
	assert.NoError(err)
	assert.Equal("baz", res.Value.Value)

	res, err = tagService.DeleteTag(authContext, req)
	assert.NoError(err)
	assert.Equal("", res.Value.Value)

	// deleting twice gives the same 204 no content but the 2nd
	// return returns an empty value
	res, err = tagService.DeleteTag(authContext, req)
	assert.NoError(err)
	assert.NotNil(res.Value)
	assert.Equal("", res.Value.Value)

	res, err = tagService.GetTag(authContext, req)
	assert.NoError(err)
	assert.Equal("", res.Value.Value)

}

// Return a dummy client that will return data elements for all queries
func newDummyDataStoreClient() datastore.DataStoreClient {
	return &dummyDataStoreClient{}
}

type dummyDataStoreClient struct {
}

func (d *dummyDataStoreClient) PutData(ctx context.Context, opts ...grpc.CallOption) (datastore.DataStore_PutDataClient, error) {
	return &dummyPutDataClient{}, nil
}

func (d *dummyDataStoreClient) GetData(ctx context.Context, in *datastore.DataFilter, opts ...grpc.CallOption) (datastore.DataStore_GetDataClient, error) {
	return &dummyGetDataClient{}, nil
}

func (d *dummyDataStoreClient) GetDataMetrics(ctx context.Context, in *datastore.DataFilter, opts ...grpc.CallOption) (*datastore.DataMetrics, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

func (d *dummyDataStoreClient) StoreData(ctx context.Context, in *datastore.DataMessage, opts ...grpc.CallOption) (*datastore.Receipt, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

type dummyClientStream struct {
}

func (dc *dummyClientStream) CloseSend() error {
	return nil
}
func (dc *dummyClientStream) Context() context.Context {
	return context.Background()
}
func (dc *dummyClientStream) Header() (metadata.MD, error) {
	return make(metadata.MD), nil
}
func (dc *dummyClientStream) SendMsg(msg interface{}) error {
	return errors.New("foo")
}
func (dc *dummyClientStream) Trailer() metadata.MD {
	return nil
}
func (dc *dummyClientStream) RecvMsg(msg interface{}) error {
	return errors.New("foo")
}

type dummyPutDataClient struct {
	dummyClientStream
}

func (dc *dummyPutDataClient) Recv() (*datastore.Receipt, error) {
	return &datastore.Receipt{}, nil
}
func (dc *dummyPutDataClient) Send(m *datastore.DataMessage) error {
	return nil
}

type dummyGetDataClient struct {
	dummyClientStream
	collectionID string
	sent         int
}

func (dc *dummyGetDataClient) RecvMsg(data interface{}) error {
	msg, ok := data.(*datastore.DataMessage)
	if !ok {
		return errors.New("foof")
	}
	if dc.sent == 10 {
		return errors.New("eof")
	}
	dc.sent++
	msg.CollectionId = dc.collectionID
	msg.DeviceId = model.DeviceKey(dc.sent).String()
	msg.Created = int64(time.Now().UnixNano() / time.Hour.Milliseconds())
	msg.Metadata = []byte(`{"transport":"udp"}`)
	msg.Payload = []byte("hello there")
	return nil
}
func (dc *dummyGetDataClient) Recv() (*datastore.DataMessage, error) {
	if dc.sent == 10 {
		return nil, errors.New("eof")
	}
	dc.sent++
	return &datastore.DataMessage{
		CollectionId: dc.collectionID,
		DeviceId:     model.DeviceKey(dc.sent).String(),
		Created:      int64(time.Now().UnixNano() / time.Hour.Milliseconds()),
		Metadata:     []byte(`{"transport":"udp"}`),
		Payload:      []byte("hello there"),
	}, nil
}

func newDummyMessageSender() *dummySender {
	return &dummySender{}
}

type dummySender struct {
	FailOnIMSI int64
}

func (d *dummySender) Send(dev model.Device, m model.DownstreamMessage) error {
	if d.FailOnIMSI == dev.IMSI {
		return errors.New("Send failed")
	}
	return nil
}
