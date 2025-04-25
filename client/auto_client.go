package client

import (
	//"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

var (
	ports = []string{"5300", "5302", "5303", "5304", "5305", "5306", "5307", "5308", "5309"}
)

func sendDNSQuery(domain string, port string, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, err := net.Dial("udp", "127.0.0.1:"+port)
	if err != nil {
		log.Printf("Failed to connect to DNS server on port %s: %v", port, err)
		return
	}
	defer conn.Close()

	/*query := []byte(domain)
	encoded := make([]byte, hex.EncodedLen(len(query)))
	hex.Encode(encoded, query)*/

	_, err = conn.Write([]byte(domain))
	if err != nil {
		log.Printf("Failed to send query: %v", err)
		return
	}

	buffer := make([]byte, 512)
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
		return
	}
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Timeout or error reading response for %s from %s", domain, port)
		return
	}

	fmt.Printf("[%s] Response for %s -> %s\n", port, domain, string(buffer[:n]))
}

func RunAutoClient(domains []string) {
	fmt.Println(" Intitiating client queries:")
	// No need for rand.Seed as we're using crypto/rand
	var wg sync.WaitGroup

	numQueries := 10

	for i := 0; i < numQueries; i++ {
		domain := domains[rand.Intn(len(domains))]
		port := ports[rand.Intn(len(ports))]

		wg.Add(1)
		go sendDNSQuery(domain, port, &wg)

		time.Sleep(300 * time.Millisecond) // brief pause between queries
	}

	wg.Wait()
	fmt.Println(" All queries completed.")
}
