package main

import (
	"fmt"
	"github.com/bleasey/bdns/internal/blockchain"
)

func main() {
	genesisBlock := blockchain.NewGenesisBlock()
    fmt.Println(genesisBlock)
}
