package master

import (
	"fmt"
	"github.com/evilsocket/islazy/log"
	. "github.com/evilsocket/sum/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"sync"
)

// information to work with a specific node
type NodeInfo struct {
	sync.RWMutex
	ID uint
	// Name of this node ( it's address )
	Name string
	// Certificate file ( TODO: optionally load it in base64 format )
	CertFile string
	// GRPC client to the node's sum service
	Client SumServiceClient
	// GRPC client to the node's sum internal service
	InternalClient SumInternalServiceClient
	// node's status
	status ServerInfo
}

// update node's status
func (n *NodeInfo) UpdateStatus() {
	n.Lock()
	defer n.Unlock()

	ctx, cf := newCommContext()
	defer cf()
	srvInfo, err := n.Client.Info(ctx, &Empty{})

	if err != nil {
		log.Error("Unable to update node '%s' status: %v", n.Name, err)
		return
	}

	n.status = *srvInfo
}

// get currently available node's status
func (n *NodeInfo) Status() ServerInfo {
	n.RLock()
	defer n.RUnlock()
	return n.status
}

// create node from a connection string and a certificate file
// this method connect to the node and create it's respective GRPC clients.
// it verifies the connection by retrieving the node's status using the aforementioned clients.
func CreateNode(node, certFile string) (*NodeInfo, error) {
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMsgSize),
			grpc.MaxCallSendMsgSize(maxMsgSize),
		),
	}

	if certFile != "" {
		creds, err := credentials.NewClientTLSFromFile(certFile, "")
		if err != nil {
			return nil, fmt.Errorf("cannot load certificate file '%s': %v", certFile, err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(node, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to dial service at '%s': %v", node, err)
	}
	client := NewSumServiceClient(conn)
	internalClient := NewSumInternalServiceClient(conn)
	ctx, cancelFn := newCommContext()
	defer cancelFn()

	// check service availability
	svcInfo, err := client.Info(ctx, &Empty{})
	if err != nil {
		return nil, fmt.Errorf("unable to get service info from node '%s': %v", node, err)
	}

	ni := &NodeInfo{
		status:         *svcInfo,
		Name:           node,
		CertFile:       certFile,
		Client:         client,
		InternalClient: internalClient,
	}

	return ni, nil
}
