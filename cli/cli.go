package cli

// 使用方法:
// 先编译出可执行文件：go build -o go-blockchain main.go
// 再通过命令行运行命令, 例如:
// 1. 添加区块:
//    ./go-blockchain addblock -data "Send 1 BTC to Ivan"
// 2. 打印区块链:
//    ./go-blockchain printchain

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/ReisenCW/go-simple-blockchain/blockchain"
)

type CLI struct {
	bc *blockchain.BlockChain
}

func NewCli(bc *blockchain.BlockChain) *CLI {
	return &CLI{bc}
}

// arg[0]是程序名
// args[1]是命令名
// args[2...]是命令参数
func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  getbalance -address ADDRESS - Get balance of ADDRESS")
	fmt.Println("  createblockchain -address ADDRESS - Create a blockchain and send genesis block reward to ADDRESS")
	fmt.Println("  printchain - Print all the blocks of the blockchain")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT - Send AMOUNT of coins from FROM address to TO")
}

func (cli *CLI) Run() {
	cli.validateArgs()

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockChainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockChainAddress := createBlockChainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	switch os.Args[1] {
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}
	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress)
	}
	if createBlockChainCmd.Parsed() {
		if *createBlockChainAddress == "" {
			createBlockChainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockChain(*createBlockChainAddress)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}

		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
}

func (cli *CLI) createBlockChain(address string) {
	bc, err := blockchain.CreateBlockChain(address)
	if err != nil {
		fmt.Printf("Error creating blockchain: %v\n", err)
		return
	}
	bc.CloseDB()
	fmt.Println("Done!")
}

func (cli *CLI) getBalance(address string) {
	bc, err := blockchain.NewBlockChain(address)
	if err != nil {
		fmt.Printf("Error creating blockchain: %v\n", err)
		return
	}
	defer bc.CloseDB()

	balance := 0
	UTXOs := bc.FindUTXO(address)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

func (cli *CLI) printChain() {
	bc, _ := blockchain.NewBlockChain("")
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
	bc, _ := blockchain.NewBlockChain(from)
	defer bc.CloseDB()

	tx, err := blockchain.NewUTXOTransaction(from, to, amount, bc)
	if err != nil {
		fmt.Printf("Failed to create transaction: %v\n", err)
		return
	}
	if err := bc.MineBlock([]*blockchain.Transaction{tx}); err != nil {
		fmt.Printf("Failed to mine block: %v\n", err)
		return
	}
	fmt.Println("Success!")
}
