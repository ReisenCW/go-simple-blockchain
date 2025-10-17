package cli

import (
	"fmt"
	"log"
	"strconv"
	"github.com/ReisenCW/go-simple-blockchain/blockchain"
)

func (cli *CLI) createBlockChain(address string) {
	if !blockchain.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc, err := blockchain.CreateBlockChain(address)
	if err != nil {
		fmt.Printf("Error creating blockchain: %v\n", err)
		return
	}
	defer bc.CloseDB()

	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	UTXOSet.Reindex()
	fmt.Println("Done!")
}

func (cli *CLI) getBalance(address string) {
	if !blockchain.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc, err := blockchain.NewBlockChain()
	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	if err != nil {
		fmt.Printf("Error creating blockchain: %v\n", err)
		return
	}
	defer bc.CloseDB()

	balance := 0
	pubKeyHash := blockchain.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

func (cli *CLI) printChain() {
	bc, _ := blockchain.NewBlockChain()
	defer bc.CloseDB()

	bci := bc.Iterator()
	for {
		block := bci.Next()

		fmt.Printf("Prev. hash: %x\n", block.PrevHash)
		fmt.Printf("Hash: %x\n", block.Hash)
		pow := blockchain.NewProofOfWork(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()

		// 遍历到头了
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CLI) send(from, to string, amount int) {
	if !blockchain.ValidateAddress(from) {
		log.Panic("ERROR: Address from is not valid")
	}
	if !blockchain.ValidateAddress(to) {
		log.Panic("ERROR: Address to is not valid")
	}
	bc, _ := blockchain.NewBlockChain()
	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	defer bc.CloseDB()

	tx, err := blockchain.NewUTXOTransaction(from, to, amount, &UTXOSet)
	cbtx := blockchain.NewCoinbaseTX(from, "")
	if err != nil {
		fmt.Printf("Failed to create transaction: %v\n", err)
		return
	}
	txs := []*blockchain.Transaction{cbtx, tx}
	newBlock := bc.MineBlock(txs)
	UTXOSet.Update(newBlock)
	fmt.Println("Success!")
}

func (cli *CLI) createWallet() {
	wallets, _ := blockchain.NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()

	fmt.Printf("Your new address: %s\n", address)
}

func (cli *CLI) listAddresses() {
	wallets, err := blockchain.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CLI) reindexUTXO() {
	bc, _ := blockchain.NewBlockChain()
	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}