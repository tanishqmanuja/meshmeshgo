package main

import (
	"context"
	"log"
	"net/http"
	"text/template"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "leguru.net/m/v2/rpc/meshmesh"
)

type TplBase struct {
	PageTitle string
	PageKey   string
}

type TplError struct {
	Error   bool
	Type    string
	Message string
}

type TplIndex struct {
	TplBase
}

type TplNode struct {
	Index uint32
	Id    uint32 `json:"id"`
	Tag   string `json:"tag"`
	InUse bool   `json:"inuse"`
}

type TplEdge struct {
	From   uint32  `json:"from"`
	To     uint32  `json:"to"`
	Weight float32 `json:"weight"`
}

type TplNetwork struct {
	TplBase
	Nodes []TplNode
	Edges []TplEdge
}

var rpcConn *grpc.ClientConn
var rpcClient pb.MeshmeshClient

func _handleError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, TplError{
		Error:   true,
		Type:    "rpc error",
		Message: err.Error(),
	})
}

func handleIndexGet(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", TplIndex{
		TplBase: TplBase{
			PageTitle: "Dashboard",
			PageKey:   "index",
		},
	})
}

func handleNetworkGet(c *gin.Context) {
	nodes, err := rpcClient.NetworkNodes(context.Background(), &pb.NetworkNodesRequest{})
	if err != nil {
		_handleError(c, err)
		return
	}
	edges, err := rpcClient.NetworkEdges(context.Background(), &pb.NetworkEdgesRequest{})
	if err != nil {
		_handleError(c, err)
		return
	}

	nodesTpl := make([]TplNode, len(nodes.Nodes))
	for i, node := range nodes.Nodes {
		nodesTpl[i] = TplNode{
			Index: uint32(i + 1),
			Id:    node.Id,
			Tag:   node.Tag,
			InUse: node.Inuse,
		}
	}

	edgesTpl := make([]TplEdge, len(edges.Edges))
	for i, edge := range edges.Edges {
		edgesTpl[i] = TplEdge{
			From:   edge.From,
			To:     edge.To,
			Weight: edge.Weight,
		}
	}

	c.HTML(http.StatusOK, "network.html", TplNetwork{
		TplBase: TplBase{
			PageTitle: "Network graph",
			PageKey:   "network",
		},
		Nodes: nodesTpl,
		Edges: edgesTpl,
	})
}

func handleEditNodePost(c *gin.Context) {
	var node TplNode
	err := c.ShouldBindJSON(&node)
	if err != nil {
		_handleError(c, err)
		return
	}

	rpcClient.NetworkNodeConfigure(context.Background(), &pb.NetworkNodeConfigureRequest{
		Id:    node.Id,
		Tag:   node.Tag,
		Inuse: node.InUse,
	})

	c.JSON(http.StatusOK, TplBase{})
}

func handleNodeDelete(c *gin.Context) {
	var node TplNode
	err := c.ShouldBindJSON(&node)
	if err != nil {
		_handleError(c, err)
		return
	}

	rpcClient.NetworkNodeDelete(context.Background(), &pb.NetworkNodeDeleteRequest{
		Id: node.Id,
	})

	c.JSON(http.StatusOK, TplBase{})
}

func initRpcClient() {
	var err error
	rpcConn, err = grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to dial rpc server: %v", err)
	}

	rpcClient = pb.NewMeshmeshClient(rpcConn)
	hello, err := rpcClient.SayHello(context.Background(), &pb.HelloRequest{})
	if err != nil {
		log.Fatalf("Failed to receive hello response")
	}
	log.Printf("Hello reply: %s, %s", hello.GetName(), hello.GetVersion())
}

func main() {
	initRpcClient()

	router := gin.Default()
	router.SetFuncMap(template.FuncMap{})
	router.LoadHTMLGlob("templates/*")
	router.Use(static.Serve("/", static.LocalFile("./static", false)))
	router.GET("/index", handleIndexGet)
	router.GET("/network", handleNetworkGet)
	router.POST("/network/node/configure", handleEditNodePost)
	router.DELETE("/network/node", handleNodeDelete)

	router.Run("localhost:8080")
	rpcConn.Close()
}
