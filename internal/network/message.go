package network

import "encoding/json"

// MessageType defines various message types in the P2P network
type MessageType string

const (
	DNSRequest       MessageType = "DNS_REQUEST"
	DNSResponse      MessageType = "DNS_RESPONSE"
	MsgTransaction   MessageType = "TRANSACTION"
	MsgBlock         MessageType = "BLOCK"
	MsgChainRequest  MessageType = "CHAIN_REQUEST"
	MsgChainResponse MessageType = "CHAIN_RESPONSE"
	MsgPeerRequest   MessageType = "PEER_REQUEST"
	MsgPeerResponse  MessageType = "PEER_RESPONSE"
	MsgRandomNumber  MessageType = "RANDOM_NUMBER"
	MsgInv           MessageType = "INV"
	MsgGetBlock      MessageType = "GET_BLOCK"
	MsgGetData       MessageType = "GET_DATA"
	MsgGetMerkle     MessageType = "GET_MERKLE"
	RegistryVoteKick MessageType = "VOTE_KICK"
)

// Message represents a generic network message
type Message struct {
	Sender string      `json:"sender"`
	Type   MessageType `json:"type"`
	Data   []byte      `json:"data"`
}

// Encode message to JSON
func (m *Message) Encode() []byte {
	data, _ := json.Marshal(m)
	return data
}

// Decode message from JSON
func DecodeMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}
