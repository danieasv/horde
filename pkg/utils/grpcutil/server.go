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
	"net"
	"time"

	"github.com/ExploratoryEngineering/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GRPCServer is the common interface for GRPC servers
type GRPCServer interface {
	// Start launches the server in the foreground
	Start(registerFunc func(s *grpc.Server)) error
	// Launch launches the server in the background
	Launch(registerFunc func(s *grpc.Server), timeout time.Duration) error
	// Endpoint returns the server's endpoint
	Endpoint() string
	// Stop shuts down the server
	Stop()
}

// NewGRPCServer configures a new GRPC server. A port will be allocated for the server
func NewGRPCServer(params GRPCServerParam) (GRPCServer, error) {
	ret := grpcServer{config: params}

	var err error
	ret.listener, err = net.Listen("tcp", ret.config.Endpoint)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

type grpcServer struct {
	config   GRPCServerParam
	listener net.Listener
	server   *grpc.Server
}

func (g *grpcServer) getOpts(config GRPCServerParam) ([]grpc.ServerOption, error) {
	if !config.TLS {
		return []grpc.ServerOption{}, nil
	}
	if config.CertFile == "" || config.KeyFile == "" {
		return nil, errors.New("missing cert file and key file parameters for GRPC server")
	}
	creds, err := credentials.NewServerTLSFromFile(config.CertFile, config.KeyFile)
	if err != nil {
		return nil, err
	}
	return []grpc.ServerOption{grpc.Creds(creds)}, nil
}

func (g *grpcServer) Start(register func(s *grpc.Server)) error {
	opts, err := g.getOpts(g.config)
	if err != nil {
		return err
	}
	g.server = grpc.NewServer(opts...)

	register(g.server)

	if err := g.server.Serve(g.listener); err != nil {
		logging.Error("Unable to serve gRPC: %v", err)
		return err
	}
	return nil
}

func (g *grpcServer) Launch(register func(s *grpc.Server), timeout time.Duration) error {
	errCh := make(chan error)

	go func() {
		if err := g.Start(register); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(timeout):
		return nil
	}
}

func (g *grpcServer) Stop() {
	if g.server != nil {
		g.server.GracefulStop()
	}
}

func (g *grpcServer) Endpoint() string {
	return g.listener.Addr().String()
}
