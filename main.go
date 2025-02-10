package main

import (
	"fmt"
	"github.com/bleasey/bdns/src/blockchain"
)

func main() {
	genesisBlock := blockchain.NewGenesisBlock()
    fmt.Println(genesisBlock)
}
