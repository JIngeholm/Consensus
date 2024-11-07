package main

import (
	leader "ConsensusGRPC/proto"
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Node struct {
	leader.UnimplementedLeaderServiceServer
	nodeID         int32
	timestamp      int64
	quorum         int
	criticalSec    bool
	peers          []string
	requests       map[int32]leader.Request
	grantMutex     sync.Mutex
	hasSentRequest bool
}

type Request struct {
	NodeID    int32
	Timestamp int64
}

func (n *Node) RequestCS(ctx context.Context, req *leader.Request) (*leader.Response, error) {

	// Log the request and update the timestamp
	fmt.Printf("Node %d requesting CS at time %s\n", req.GetNodeId(), unixNanoToDateString(req.GetTimestamp()))

	if req.GetTimestamp() > n.timestamp {
		n.timestamp = req.GetTimestamp()
	}
	n.timestamp++
	n.hasSentRequest = true

	// Send requests to other nodes
	var replies int
	for _, peer := range n.peers {
		if peer != fmt.Sprintf("localhost:%d", n.nodeID) {
			fmt.Printf("Asking %s for reply with request time: %s", peer, unixNanoToDateString(req.GetTimestamp()))
			fmt.Println()
			client, err := n.getClient(peer)
			if err != nil {
				log.Fatal(err)
			}

			resp, err := client.ReplyCS(ctx, req)
			if err != nil {
				log.Fatal(err)
			}

			if resp.GetSuccess() {
				fmt.Printf("Got a successful reply from %s", peer)
				fmt.Println()
				replies++
			} else {
				fmt.Printf("Did not get a successful reply from %s", peer)
				fmt.Println()

			}
		}
	}

	// If quorum is reached, grant access to the CS
	if replies >= n.quorum {
		n.criticalSec = true
		fmt.Printf("Node %d has entered the Critical Section.\n", n.nodeID)

		// Simulate accessing the Critical Section
		n.enterCS()
		n.timestamp = time.Now().UnixNano()
		n.criticalSec = false

	}
	bestTime := math.MaxInt
	var bestReq leader.Request
	for _, req := range n.requests {
		if req.GetTimestamp() < int64(bestTime) {
			bestTime = int(req.GetTimestamp())
			bestReq = req
		}

	}
	client, err := n.getClient(fmt.Sprintf("localhost:%d", bestReq.NodeId))
	if err != nil {
		return nil, err
	}
	client.RequestCS(context.Background(), &bestReq)
	for k := range n.requests {
		delete(n.requests, k)
	}

	return &leader.Response{Success: true}, nil
}

func (n *Node) ReplyCS(ctx context.Context, req *leader.Request) (*leader.Response, error) {

	// Handle reply logic based on Ricart-Agrawala
	fmt.Printf("Comparing %d's req timestamp %s to own timestamp %s", req.NodeId, unixNanoToDateString(req.GetTimestamp()), unixNanoToDateString(n.timestamp))
	fmt.Println()
	if req.GetTimestamp() <= n.timestamp || !n.hasSentRequest {
		fmt.Println("Success")
		return &leader.Response{Success: true}, nil
	}
	n.requests[req.GetNodeId()] = *req

	fmt.Println("Failure")
	return &leader.Response{Success: false}, nil
}

func (n *Node) getClient(server string) (leader.LeaderServiceClient, error) {
	conn, err := grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return leader.NewLeaderServiceClient(conn), nil
}

func (n *Node) enterCS() {
	// Simulate the critical section
	fmt.Printf("Node %d is accessing the Critical Section\n", n.nodeID)
	time.Sleep(10 * time.Second) // Simulate work in CS
	fmt.Printf("Node %d is leaving the Critical Section\n", n.nodeID)
}

func (n *Node) startServer() {
	server := grpc.NewServer()
	leader.RegisterLeaderServiceServer(server, n)

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", n.nodeID))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	fmt.Printf("Node %d started on port %d\n", n.nodeID, n.nodeID)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func main() {
	// Example: Node 1 running on port 50051
	port := flag.Int64("port", 50051, "Port for the gRPC server to listen on")
	flag.Parse()

	node := &Node{
		nodeID:    int32(*port),
		timestamp: time.Now().UnixNano(),
		quorum:    2, // Assuming we have 3 nodes, quorum is 2
		peers:     []string{"localhost:50051", "localhost:50052", "localhost:50053"},
		requests:  make(map[int32]leader.Request),
	}

	go node.startServer()

	// Simulate sending a request to enter the Critical Section
	time.Sleep(10 * time.Second)

	// Make the request (you can add more logic for when nodes send requests)
	client, err := node.getClient(fmt.Sprintf("localhost:%d", node.nodeID))
	if err != nil {
		log.Fatal(err)
	}

	req := &leader.Request{
		NodeId:    node.nodeID,
		Timestamp: time.Now().UnixNano(),
	}

	// Send request to other nodes
	_, err = client.RequestCS(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	select {}
}

func unixNanoToDateString(unixNano int64) string {
	// Convert UnixNano to a time.Time object
	t := time.Unix(0, unixNano)
	// Format the time as a string
	return t.Format("2006-01-02 15:04:05.999999999")
}
