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

func (s *Server) EntitiesCount(_ context.Context, req *meshmesh.EntitiesCountRequest) (*meshmesh.EntitiesCountReply, error) {
	rep, err := s.serialConn.SendReceiveApiProt(mm.EntitiesCountApiRequest{}, mm.UnicastProtocol, mm.MeshNodeId(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get entities count: %v", err)
	}
	cnt := rep.(mm.EntitiesCountApiReply)
	return &meshmesh.EntitiesCountReply{
		All:           uint32(cnt.Counters[0]),
		Sensors:       uint32(cnt.Counters[1]),
		BinarySensors: uint32(cnt.Counters[2]),
		Switches:      uint32(cnt.Counters[3]),
		Lights:        uint32(cnt.Counters[4]),
		TextSensors:   uint32(cnt.Counters[5]),
	}, nil
}

func (s *Server) EntityHash(_ context.Context, req *meshmesh.EntityHashRequest) (*meshmesh.EntityHashReply, error) {
	rep, err := s.serialConn.SendReceiveApiProt(mm.EntityHashApiRequest{Service: uint8(req.Service), Index: uint8(req.Index)}, mm.UnicastProtocol, mm.MeshNodeId(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get entity hash: %v", err)
	}
	hash := rep.(mm.EntityHashApiReply)
	if hash.Hash == 0 && hash.Info == "E!" {
		return nil, status.Errorf(codes.NotFound, "Entity not found")
	}
	return &meshmesh.EntityHashReply{Id: req.Id, Hash: uint32(hash.Hash), Info: hash.Info}, nil
}

func (s *Server) GetEntityState(_ context.Context, req *meshmesh.GetEntityStateRequest) (*meshmesh.GetEntityStateReply, error) {
	rep, err := s.serialConn.SendReceiveApiProt(mm.GetEntityStateApiRequest{
		Service: uint8(req.Service),
		Hash:    uint16(req.Hash),
	}, mm.UnicastProtocol, mm.MeshNodeId(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get entity state: %v", err)
	}
	state := rep.(mm.GetEntityStateApiReply)
	return &meshmesh.GetEntityStateReply{State: uint32(state.State)}, nil
}

func (s *Server) SetEntityState(_ context.Context, req *meshmesh.SetEntityStateRequest) (*meshmesh.SetEntityStateReply, error) {
	_, err := s.serialConn.SendReceiveApiProt(mm.SetEntityStateApiRequest{
		Service: uint8(req.Service),
		Hash:    uint16(req.Hash),
		State:   uint16(req.State),
	}, mm.UnicastProtocol, mm.MeshNodeId(req.Id))

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to set entity state: %v", err)
	}
	return &meshmesh.SetEntityStateReply{Success: true}, nil
}

func (s *Server) NetworkNodes(_ context.Context, req *meshmesh.NetworkNodesRequest) (*meshmesh.NetworkNodesReply, error) {
	nodes := s.network.Nodes()
	device := make([]*meshmesh.NetworkNode, nodes.Len())
	i := 0
	for nodes.Next() {
		dev := nodes.Node().(*graph.Device)
		device[i] = &meshmesh.NetworkNode{
			Id:    uint32(dev.ID()),
			Tag:   string(dev.Tag()),
			Inuse: dev.InUse(),
		}
		i += 1
	}
	return &meshmesh.NetworkNodesReply{Nodes: device}, nil
}

func (s *Server) NetworkEdges(_ context.Context, req *meshmesh.NetworkEdgesRequest) (*meshmesh.NetworkEdgesReply, error) {
	edges := s.network.WeightedEdges()
	_edges := make([]*meshmesh.NetworkEdge, edges.Len())
	i := 0
	for edges.Next() {
		edge := edges.WeightedEdge()
		_edges[i] = &meshmesh.NetworkEdge{
			From:   uint32(edge.From().ID()),
			To:     uint32(edge.To().ID()),
			Weight: float32(edge.Weight()),
		}
		i += 1
	}
	return &meshmesh.NetworkEdgesReply{Edges: _edges}, nil
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
