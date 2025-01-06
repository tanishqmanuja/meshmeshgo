package rpc

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"leguru.net/m/v2/graph"
	"leguru.net/m/v2/logger"
	mm "leguru.net/m/v2/meshmesh"
	"leguru.net/m/v2/rpc/meshmesh"
)

// protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative meshmesh/meshmesh.proto

type Server struct {
	meshmesh.UnimplementedMeshmeshServer
	serialConn     *mm.SerialConnection
	network        *graph.Network
	programName    string
	programVersion string
}

func NewServer(programName string, programVersion string, serialConn *mm.SerialConnection, network *graph.Network) *Server {
	return &Server{programName: programName, programVersion: programVersion, serialConn: serialConn, network: network}
}

func (s *Server) SayHello(_ context.Context, req *meshmesh.HelloRequest) (*meshmesh.HelloReply, error) {
	return &meshmesh.HelloReply{Name: s.programName, Version: s.programVersion}, nil
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

func (s *RpcServer) Start(programName string, programVersion string, serialConn *mm.SerialConnection, network *graph.Network) error {
	var err error
	s.lis, err = net.Listen("tcp", s.port)
	if err != nil {
		return err
	}

	s.grpcServer = grpc.NewServer()
	meshmesh.RegisterMeshmeshServer(s.grpcServer, NewServer(programName, programVersion, serialConn, network))
	logger.WithField("port", s.port).Info("Starting gRPC server")
	go s.serve()

	return nil
}

func (s *RpcServer) Stop() {
	s.grpcServer.Stop()
}
