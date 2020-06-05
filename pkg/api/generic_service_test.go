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

	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Generic testing of requests. All requests should be authenticated, return
// appropriate InvalidArgument responses on missing or incorrect parameters
// and all of these tests are quite repetitive. This is a bit convoluted but
// it makes it possible to test those features without too much boilerplate
// code.

type requestFunc func(context.Context, interface{}) (interface{}, error)

// Factor to ease testing the defaults for requests. All requests should contain
// an OutputId and a CollectionId property. It makes the testing quite a bit
// more opaque but saves us a lot of typing. The factory object creates the
// request object and the requestMethod type will invoke the method on the
// service object
type requestFactory interface {
	// Return a valid request (excl the collection and identifier ID)
	ValidRequest() interface{}
	// Set the collection ID
	SetCollection(req interface{}, cid *wrappers.StringValue)
	// Set the identifier ID (output ID, device ID, firmware ID...)
	SetIdentifier(req interface{}, oid *wrappers.StringValue)
}

type tparam struct {
	Assert                    *require.Assertions
	AuthenticatedContext      context.Context
	CollectionID              string
	IdentifierID              string
	RequestFunc               requestFunc
	RequestFactory            requestFactory
	TestWithInvalidIdentifier bool
}

// Note: ctx is an authenticated context, the invalidIdentifier is set to true
// if tests with the identifier should be performed
func genericRequestTests(p tparam) {
	// Nil parameters => error
	_, err := p.RequestFunc(p.AuthenticatedContext, nil)
	p.Assert.Error(err)
	p.Assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	// Nil collection id + output id => error
	req := p.RequestFactory.ValidRequest()
	p.RequestFactory.SetCollection(req, nil)
	p.RequestFactory.SetIdentifier(req, nil)
	_, err = p.RequestFunc(p.AuthenticatedContext, req)
	p.Assert.Error(err)
	p.Assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	p.RequestFactory.SetIdentifier(req, &wrappers.StringValue{Value: p.IdentifierID})
	_, err = p.RequestFunc(p.AuthenticatedContext, req)
	p.Assert.Error(err)
	p.Assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	p.RequestFactory.SetCollection(req, &wrappers.StringValue{Value: p.CollectionID})
	if p.TestWithInvalidIdentifier {
		p.RequestFactory.SetIdentifier(req, nil)
		_, err = p.RequestFunc(p.AuthenticatedContext, req)
		p.Assert.Error(err)
		p.Assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	}

	p.RequestFactory.SetIdentifier(req, &wrappers.StringValue{Value: p.IdentifierID})

	// No auth context => error
	_, err = p.RequestFunc(context.Background(), req)
	p.Assert.Error(err)
	p.Assert.Equal(codes.Unauthenticated.String(), status.Code(err).String())

	// Auth context, parameters OK => result
	res, err := p.RequestFunc(p.AuthenticatedContext, req)
	p.Assert.NoError(err)
	p.Assert.NotNil(res)

	if p.TestWithInvalidIdentifier {
		// Unknown output ID => error
		p.RequestFactory.SetIdentifier(req, &wrappers.StringValue{Value: "0"})
		_, err = p.RequestFunc(p.AuthenticatedContext, req)
		p.Assert.Error(err)
		p.Assert.Equal(codes.NotFound.String(), status.Code(err).String())
	}

	// Invalid collection ID => error
	p.RequestFactory.SetCollection(req, &wrappers.StringValue{Value: ". 1 ."})
	_, err = p.RequestFunc(p.AuthenticatedContext, req)
	p.Assert.Error(err)
	p.Assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())

	if p.TestWithInvalidIdentifier {
		// Invalid output ID => error
		p.RequestFactory.SetCollection(req, &wrappers.StringValue{Value: "0"})
		p.RequestFactory.SetIdentifier(req, &wrappers.StringValue{Value: ". 2 ."})
		_, err = p.RequestFunc(p.AuthenticatedContext, req)
		p.Assert.Error(err)
		p.Assert.Equal(codes.InvalidArgument.String(), status.Code(err).String())
	}
}
