package main

import (
	"github.com/ReisenCW/go-simple-blockchain/blockchain"
	"github.com/ReisenCW/go-simple-blockchain/cli"
)

func main() {
	bc, _ := blockchain.NewBlockChain()
	defer bc.CloseDB()

	cli := cli.NewCli(bc)
	cli.Run()
}