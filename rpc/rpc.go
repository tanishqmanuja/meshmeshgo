package rpc

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"leguru.net/m/v2/logger"
	"leguru.net/m/v2/rpc/meshmesh"
)

type Server struct {
	meshmesh.UnimplementedMeshmeshServer
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) SayHello(_ context.Context, req *meshmesh.HelloRequest) (*meshmesh.HelloReply, error) {
	return &meshmesh.HelloReply{Message: "Hello, " + req.Name}, nil
}

type RpcServer struct {
	port       string
	lis        net.Listener
	grpcServer *grpc.Server
}

func NewRpcServer(port string) *RpcServer {
	return &RpcServer{port: port}
}

func (s *RpcServer) serve() {
	if err := s.grpcServer.Serve(s.lis); err != nil {
		logger.WithField("err", err).Error("Failed to serve gRPC server")
	}
}

func (s *RpcServer) Start() error {
	var err error
	s.lis, err = net.Listen("tcp", s.port)
	if err != nil {
		return err
	}

	s.grpcServer = grpc.NewServer()
	meshmesh.RegisterMeshmeshServer(s.grpcServer, NewServer())
	logger.WithField("port", s.port).Info("Starting gRPC server")
	go s.serve()

	return nil
}

func (s *RpcServer) Stop() {
	s.grpcServer.Stop()
}
