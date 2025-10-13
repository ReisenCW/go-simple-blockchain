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
	fmt.Println("  addblock -data BLOCK_DATA - add a block to the blockchain")
	fmt.Println("  printchain - print all the blocks of the blockchain")
}

func (cli *CLI) Run() {
	cli.validateArgs()

	addBlockCmd := flag.NewFlagSet("addblock", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)

	// 	为addblock命令注册一个参数选项-data：
	// 第一个参数"data"：选项名（用户输入时需用-data指定）。
	// 第二个参数""：data的默认值（若用户未指定-data，则为为空字符串）。
	// 第三个参数"Block data"：选项描述（用于-h/--help时显示帮助信息）。
	// 返回值addBlockData是一个字符串指针，后续可通过*addBlockData获取用户输入的区块数据。
	addBlockData := addBlockCmd.String("data", "", "Block data")

	switch os.Args[1] {
	case "addblock":
		addBlockCmd.Parse(os.Args[2:])
	case "printchain":
		printChainCmd.Parse(os.Args[2:])
	default:
		cli.printUsage()
		os.Exit(1)
	}

	// 判断addblock命令是否被成功解析
	if addBlockCmd.Parsed() {
		// 检查数据是否为空, 若为空，提示用法
		if *addBlockData == "" { 
			addBlockCmd.Usage()
			os.Exit(1)
		}
		cli.addBlock(*addBlockData)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}
}

func (cli *CLI) addBlock(data string) {
	cli.bc.AddBlock(data)
	fmt.Println("Success!")
}

func (cli *CLI) printChain() {
	bci := cli.bc.Iterator()

	for {
		block := bci.Next()

		fmt.Printf("Prev. hash: %x\n", block.PrevHash)
		fmt.Printf("Data: %s\n", block.Data)
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