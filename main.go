package main

import(
	"github.com/ReisenCW/go-simple-blockchain/blockchain"
)

func main() {
	bc := blockchain.NewBlockChain()
	bc.AddBlock("First Block")
	bc.AddBlock("Second Block")
	bc.PrintBlockChain()
}
