package network

import (
	//"encoding/hex"
	"fmt"
	"log"
	"net"
)

// StartDNSServer initializes and starts a UDP-based DNS server
func StartDNSServer(port string, node *Node) {
	addr := fmt.Sprintf(":%s", port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Failed to start UDP server: %v", err)
	}
	defer conn.Close()

	//log.Printf("BDNS Server started on port %s...\n", port)

	buffer := make([]byte, 512)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Failed to read UDP request: %v", err)
			continue
		}

		query := string(buffer[:n])
		log.Printf("Received query: %s", query)

		response, err := ResolveDomain(query, node)
		if err != nil {
			log.Printf("Resolution error: %v", err)
			response = "ERROR: Domain not found"
		}

		_, err = conn.WriteToUDP([]byte(response), clientAddr)
		if err != nil {
			log.Printf("Failed to send response: %v", err)
		}
	}
}
