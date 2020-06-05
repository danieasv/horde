package grpcutil
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
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func getDialOpts(config GRPCClientParam) ([]grpc.DialOption, error) {
	if !config.TLS {
		return []grpc.DialOption{grpc.WithInsecure()}, nil
	}

	if config.CAFile == "" {
		return nil, errors.New("missing CA file for TLS")
	}

	creds, err := credentials.NewClientTLSFromFile(config.CAFile, config.ServerHostOverride)
	if err != nil {
		return nil, err
	}
	return []grpc.DialOption{grpc.WithTransportCredentials(creds)}, nil
}

// NewGRPCClientConnection is a factory method to create gRPC client connections
func NewGRPCClientConnection(config GRPCClientParam) (*grpc.ClientConn, error) {
	opts, err := getDialOpts(config)
	if err != nil {
		return nil, err
	}

	return grpc.Dial(config.ServerEndpoint, opts...)
}
