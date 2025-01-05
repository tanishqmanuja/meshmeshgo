package rpc

import (
	"bytes"
	"context"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (s *Server) NodeInfo(_ context.Context, req *meshmesh.NodeInfoRequest) (*meshmesh.NodeInfoReply, error) {
	rep, err := s.serialConn.SendReceiveApiProt(mm.FirmRevApiRequest{}, mm.UnicastProtocol, mm.MeshNodeId(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get firmware revision: %v", err)
	}
	rev := rep.(mm.FirmRevApiReply)

	rep, err = s.serialConn.SendReceiveApiProt(mm.NodeConfigApiRequest{}, mm.UnicastProtocol, mm.MeshNodeId(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get node configuration: %v", err)
	}
	cfg := rep.(mm.NodeConfigApiReply)

	return &meshmesh.NodeInfoReply{
		Id:      req.Id,
		Tag:     string(cfg.Tag[:bytes.IndexByte(cfg.Tag, 0)]),
		Channel: uint32(cfg.Channel),
		Rev:     rev.Revision[:strings.IndexByte(rev.Revision, 0)]}, nil
}

func (s *Server) NodeReboot(_ context.Context, req *meshmesh.NodeRebootRequest) (*meshmesh.NodeRebootReply, error) {
	_, err := s.serialConn.SendReceiveApiProt(mm.NodeRebootApiRequest{}, mm.UnicastProtocol, mm.MeshNodeId(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to reboot node: %v", err)
	}
	return &meshmesh.NodeRebootReply{Success: true}, nil
}

func (s *Server) BindClear(_ context.Context, req *meshmesh.BindClearRequest) (*meshmesh.BindClearReply, error) {
	_, err := s.serialConn.SendReceiveApiProt(mm.NodeBindClearApiRequest{}, mm.UnicastProtocol, mm.MeshNodeId(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to clear binded server: %v", err)
	}
	return &meshmesh.BindClearReply{Success: true}, nil
}

func (s *Server) SetTag(_ context.Context, req *meshmesh.SetTagRequest) (*meshmesh.SetTagReply, error) {
	if len(req.Tag) > 30 {
		return nil, status.Errorf(codes.InvalidArgument, "Tag must be less than 30 characters")
	}
	_, err := s.serialConn.SendReceiveApiProt(mm.NodeSetTagApiRequest{Tag: req.Tag}, mm.UnicastProtocol, mm.MeshNodeId(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to set tag: %v", err)
	}
	return &meshmesh.SetTagReply{Success: true}, nil
}

func (s *Server) SetChannel(_ context.Context, req *meshmesh.SetChannelRequest) (*meshmesh.SetChannelReply, error) {
	if req.Channel < 1 || req.Channel > 13 {
		return nil, status.Errorf(codes.InvalidArgument, "Channel must be between 1 and 13")
	}
	_, err := s.serialConn.SendReceiveApiProt(mm.NodeSetChannelApiRequest{Channel: uint8(req.Channel)}, mm.UnicastProtocol, mm.MeshNodeId(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to set channel: %v", err)
	}
	return &meshmesh.SetChannelReply{Success: true}, nil
}

func (s *Server) NetworkNodes(_ context.Context, req *meshmesh.NetworkNodesRequest) (*meshmesh.NetworkNodesReply, error) {
	nodes := s.network.Nodes()
	ids := make([]uint32, nodes.Len())
	i := 0
	for nodes.Next() {
		node := nodes.Node()
		ids[i] = uint32(node.ID())
		i += 1
	}
	return &meshmesh.NetworkNodesReply{Nodes: ids}, nil
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
