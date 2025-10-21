package cli

import (
	"fmt"
	"log"
	"strconv"
	"github.com/ReisenCW/go-simple-blockchain/blockchain"
)

func (cli *CLI) createBlockChain(address string, nodeID string) {
	if !blockchain.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc, err := blockchain.CreateBlockChain(address, nodeID)
	if err != nil {
		fmt.Printf("Error creating blockchain: %v\n", err)
		return
	}
	defer bc.CloseDB()

	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	UTXOSet.Reindex()
	fmt.Println("Done!")
}

func (cli *CLI) getBalance(address string, nodeID string) {
	if !blockchain.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc, err := blockchain.NewBlockChain(nodeID)
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

func (cli *CLI) printChain(nodeID string) {
	bc, _ := blockchain.NewBlockChain(nodeID)
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

func (cli *CLI) send(from, to string, amount int, nodeID string, mineNow bool) {
	if !blockchain.ValidateAddress(from) {
		log.Panic("ERROR: Address from is not valid")
	}
	if !blockchain.ValidateAddress(to) {
		log.Panic("ERROR: Address to is not valid")
	}
	bc, _ := blockchain.NewBlockChain(nodeID)
	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	defer bc.CloseDB()

	wallets, err := blockchain.NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)

	tx, err := blockchain.NewUTXOTransaction(&wallet, to, amount, &UTXOSet)
	if err != nil {
		log.Panic(err)
	}

	if mineNow {
		cbTx := blockchain.NewCoinbaseTX(from, "")
		txs := []*blockchain.Transaction{cbTx, tx}

		newBlock := bc.MineBlock(txs)
		UTXOSet.Update(newBlock)
	} else {
		blockchain.SendTx(blockchain.GetCentralNodeAddress(), tx)
	}
	fmt.Println("Success!")
}

func (cli *CLI) createWallet(nodeID string) {
	wallets, _ := blockchain.NewWallets(nodeID)
	address := wallets.CreateWallet()
	wallets.SaveToFile(nodeID)

	fmt.Printf("Your new address: %s\n", address)
}

func (cli *CLI) listAddresses(nodeID string) {
	wallets, err := blockchain.NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CLI) reindexUTXO(nodeID string) {
	bc, _ := blockchain.NewBlockChain(nodeID)
	UTXOSet := blockchain.UTXOSet{Blockchain: bc}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)
}

func (cli *CLI) startNode(nodeID, minerAddress string) {
	fmt.Printf("Starting node %s\n", nodeID)
	if len(minerAddress) > 0 {
		if blockchain.ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}
	blockchain.StartServer(nodeID, minerAddress)
}