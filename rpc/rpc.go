package rpc

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"leguru.net/m/v2/logger"
	mm "leguru.net/m/v2/meshmesh"
	"leguru.net/m/v2/rpc/meshmesh"
)

// protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative meshmesh/meshmesh.proto

type Server struct {
	meshmesh.UnimplementedMeshmeshServer
	serialConn     *mm.SerialConnection
	programName    string
	programVersion string
}

func NewServer(programName string, programVersion string, serialConn *mm.SerialConnection) *Server {
	return &Server{programName: programName, programVersion: programVersion, serialConn: serialConn}
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

func (s *RpcServer) Start(programName string, programVersion string, serialConn *mm.SerialConnection) error {
	var err error
	s.lis, err = net.Listen("tcp", s.port)
	if err != nil {
		return err
	}

	s.grpcServer = grpc.NewServer()
	meshmesh.RegisterMeshmeshServer(s.grpcServer, NewServer(programName, programVersion, serialConn))
	logger.WithField("port", s.port).Info("Starting gRPC server")
	reflection.Register(s.grpcServer)
	go s.serve()

	return nil
}

func (s *RpcServer) Stop() {
	s.grpcServer.Stop()
}
