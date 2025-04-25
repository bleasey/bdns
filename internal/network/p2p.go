package network

import (
	"context"
	"encoding/json"
	"log"

	"github.com/bleasey/bdns/internal/metrics"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// P2PNetwork struct manages libp2p host and pubsub
type P2PNetwork struct {
	Host    host.Host
	PubSub  *pubsub.PubSub
	Topic   *pubsub.Topic
	Sub     *pubsub.Subscription
	MsgChan chan GossipMessage
}

// GossipMessage structure
type GossipMessage struct {
	Type    MessageType
	Sender  string // Peer ID
	Content json.RawMessage
	Metrics *metrics.DNSMetrics
}

// NewP2PNetwork initializes a libp2p node with pubsub gossip
func NewP2PNetwork(ctx context.Context, addr string, topicName string) (*P2PNetwork, error) {
	// Create libp2p host
	h, err := libp2p.New(libp2p.ListenAddrStrings(addr))
	if err != nil {
		return nil, err
	}

	// Initialize pubsub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}

	// Join gossip topic
	topic, err := ps.Join(topicName)
	if err != nil {
		return nil, err
	}

	// Subscribe to the topic
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	network := &P2PNetwork{
		Host:    h,
		PubSub:  ps,
		Topic:   topic,
		Sub:     sub,
		MsgChan: make(chan GossipMessage),
	}

	return network, nil
}

// ListenForGossip listens for gossip messages
func (p *P2PNetwork) ListenForGossip() {
	for {
		msg, err := p.Sub.Next(context.Background())
		if err != nil {
			log.Println("Error reading from subscription:", err)
			continue
		}

		var gMsg GossipMessage
		err = json.Unmarshal(msg.Data, &gMsg)
		if err != nil {
			log.Println("Error decoding gossip message:", err)
			continue
		}

		// Ignore messages sent by itself
		if gMsg.Sender == p.Host.ID().String() {
			continue
		}

		p.MsgChan <- gMsg
	}
}

// BroadcastMessage publishes a message to the gossip network
func (p *P2PNetwork) BroadcastMessage(msgType MessageType, content interface{}, metrics *metrics.DNSMetrics) {
	data, _ := json.Marshal(content)
	gossipMsg := GossipMessage{
		Type:    msgType,
		Sender:  p.Host.ID().String(),
		Content: data,
		Metrics: metrics,
	}

	msgData, _ := json.Marshal(gossipMsg)
	if err := p.Topic.Publish(context.Background(), msgData); err != nil {
		log.Printf("failed to publish message: %v", err)
	}
}

// DirectMessage sends a message to a specific peer
func (p *P2PNetwork) DirectMessage(msgType MessageType, content interface{}, peerIDStr string) {
	peerID, err := peer.Decode(peerIDStr)
	if err != nil {
		log.Println("Invalid Peer ID:", err)
		return
	}

	stream, err := p.Host.NewStream(context.Background(), peerID, "/dns-response")
	if err != nil {
		log.Println("Failed to open stream:", err)
		return
	}
	defer stream.Close()

	data, _ := json.Marshal(content)
	responseMsg := GossipMessage{
		Type:    msgType,
		Sender:  p.Host.ID().String(),
		Content: data,
	}

	enc := json.NewEncoder(stream)
	if err := enc.Encode(responseMsg); err != nil {
		log.Printf("failed to encode message: %v", err)
	}
}

// ConnectToPeer connects to another peer via multiaddress
func (p *P2PNetwork) ConnectToPeer(peerAddr string) error {
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return err
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	return p.Host.Connect(context.Background(), *peerInfo)
}

// Close shuts down the network
func (p *P2PNetwork) Close() {
	p.Host.Close()
}
