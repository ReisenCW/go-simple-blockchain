package blockchain

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
)

const protocol = "tcp"
const nodeVersion = 1
const commandLength = 12

var nodeAddress string		// 当前节点的网络地址
var miningAddress string	// 挖矿奖励接收地址
var knownNodes = []string{"localhost:3000"}
var blocksInTransit = [][]byte{}	
var mempool = make(map[string]Transaction)

// 版本信息
// AddrFrom		发送该信息的节点地址
// BestHeight	该节点的区块链最高高度
// Version		节点版本号
type verzion struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

// 地址
// AddrList		地址列表
type addr struct {
	AddrList []string
}

// 发送区块
// AddrFrom		发送该区块的节点地址
// Block		序列化后的区块数据
type block struct {
	AddrFrom string
	Block    []byte
}

// 获取区块请求
// AddrFrom		发送该信息的节点地址
type getblocks struct {
	AddrFrom string
}

// 获取数据请求
// AddrFrom		发送该信息的节点地址
// Type			信息类型，区块("block")或交易("tx")
// ID			块的哈希或交易ID
type getdata struct {
	AddrFrom string
	Type     string
	ID       []byte
}

// 使用inv向其他节点展示当前节点有什么块/交易
// AddrFrom		发送该信息的节点地址
// Type			信息类型，区块("block")或交易("tx")
// Items		块的哈希列表或交易ID列表
type inv struct {
    AddrFrom string
    Type     string
    Items    [][]byte
}

// 发送交易
// AddrFrom		发送该信息的节点地址
// Transaction		序列化后的交易数据
type tx struct {
	AddrFrom     string
	Transaction  []byte
}

// 获取中央节点的地址
func GetCentralNodeAddress() string {
	return knownNodes[0]
}

// 启动节点服务器，监听来自其他节点的连接请求
func StartServer(nodeID, minerAddress string) {
	// 根据端口号(nodeID)设置当前节点的网络地址
    nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	// 设置挖矿奖励接收地址
    miningAddress = minerAddress
	// 监听网络连接
    ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
    defer ln.Close()

    bc, _ := NewBlockChain(nodeID)

	// 如果当前节点不是中央节点，向中央节点发送版本信息
    if nodeAddress != GetCentralNodeAddress() {
		// 通常用于节点间同步区块链版本（如区块高度、最新区块哈希等），目的是让当前节点与种子节点对齐账本状态（如果本地区块链落后，会触发区块同步）
        SendVersion(GetCentralNodeAddress(), bc)
    }

	// 不断接受并处理来自其他节点的连接请求
    for {
        conn, _ := ln.Accept()
		// 启动一个 goroutine(轻量级线程)异步执行，而不会阻塞当前的主程序流程
        go handleConnection(conn, bc)
    }
}

// 发送数据给指定地址的节点
func SendData(addr string, data []byte) {
	// 建立与目标节点的网络连接, protocol = "tcp"
	conn, err := net.Dial(protocol, addr)
	// 连接失败则说明该节点不可用
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string
		// 从已知节点列表中移除该不可用节点
		for _, node := range knownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}
		knownNodes = updatedNodes
		return
	}
	defer conn.Close()

	// 发送数据
	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

// 发送信息给指定地址的节点
func SendVersion(addr string, bc *BlockChain) {
    bestHeight := bc.GetBestHeight()
    payload := gobEncode(verzion{nodeVersion, bestHeight, nodeAddress})

    request := append(commandToBytes("version"), payload...)
    SendData(addr, request)
}

// 发送区块给指定地址的节点
func SendBlock(addr string, b *Block) {
	data := block{nodeAddress, b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("block"), payload...)

	SendData(addr, request)
}

// 发送inv信息给指定地址的节点
func SendInv(address, kind string, items [][]byte) {
	inventory := inv{nodeAddress, kind, items}
	payload := gobEncode(inventory)
	request := append(commandToBytes("inv"), payload...)

	SendData(address, request)
}

// 发送交易给指定地址的节点
func SendTx(addr string, tnx *Transaction) {
	data := tx{nodeAddress, tnx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)

	SendData(addr, request)
}

// 发送获取区块请求给指定地址的节点
func SendGetBlocks(address string) {
	payload := gobEncode(getblocks{nodeAddress})
	request := append(commandToBytes("getblocks"), payload...)

	SendData(address, request)
}

// 发送获取数据请求给指定地址的节点
func SendGetData(address, kind string, id []byte) {
	payload := gobEncode(getdata{nodeAddress, kind, id})
	request := append(commandToBytes("getdata"), payload...)

	SendData(address, request)
}

// 将命令字符串转换为固定长度的字节数组
func commandToBytes(command string) []byte {
    var bytes [commandLength]byte

    for i, c := range command {
        bytes[i] = byte(c)
    }

    return bytes[:]
}

// 将字节数组转换回命令字符串
func bytesToCommand(bytes []byte) string {
    var command []byte

    for _, b := range bytes {
        if b != 0x0 {
            command = append(command, b)
        }
    }

    return fmt.Sprintf("%s", command)
}

// 提取请求中的命令部分
func extractCommand(request []byte) []byte {
	return request[:commandLength]
}

// 所有已知节点向中心节点请求区块列表, 以便同步区块链
func requestBlocks() {
	for _, node := range knownNodes {
		SendGetBlocks(node)
	}
}

// 处理来自其他节点的连接请求
func handleConnection(conn net.Conn, bc *BlockChain) {
	// 读取连接中的数据
    request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	// 提取命令
    command := bytesToCommand(request[:commandLength])
    fmt.Printf("Received %s command\n", command)

	// 处理不同类型的命令
	switch command {
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getblocks":
		handleGetBlocks(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)
	case "version":
		handleVersion(request, bc)
	default:
		fmt.Println("Unknown command!")
	}

    conn.Close()
}

// 处理接收到的版本信息
func handleVersion(request []byte, bc *BlockChain) {
    var buff bytes.Buffer
    var payload verzion

    buff.Write(request[commandLength:])
    dec := gob.NewDecoder(&buff)
    dec.Decode(&payload)

    myBestHeight := bc.GetBestHeight()
    foreignerBestHeight := payload.BestHeight

    if myBestHeight < foreignerBestHeight {
        SendGetBlocks(payload.AddrFrom)
    } else if myBestHeight > foreignerBestHeight {
        SendVersion(payload.AddrFrom, bc)
    }

    if !nodeIsKnown(payload.AddrFrom) {
        knownNodes = append(knownNodes, payload.AddrFrom)
    }
}

// 处理接收到的地址信息
func handleAddr(request []byte) {
	var buff bytes.Buffer
	var payload addr

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	knownNodes = append(knownNodes, payload.AddrList...)
	fmt.Printf("There are %d known nodes now!\n", len(knownNodes))
	requestBlocks()
}

// 处理获取区块请求
func handleGetBlocks(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload getblocks

	// 读取请求中的负载数据
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := bc.GetBlockHashes()
	SendInv(payload.AddrFrom, "block", blocks)
}

// 处理获取数据请求
func handleGetData(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload getdata

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	// 根据请求类型发送相应的数据
	// 请求"块"
	if payload.Type == "block" {
		block, err := bc.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}
		SendBlock(payload.AddrFrom, &block)
	}
	// 请求"交易"
	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := mempool[txID]

		SendTx(payload.AddrFrom, &tx)
	}
}

// 处理接接收到的区块
func handleBlock(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	block := DeserializeBlock(blockData)

	fmt.Println("Recevied a new block!")
	// 将接收到的区块添加到本地区块链
	bc.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)

	// 如果还有待下载的块，继续请求下一个块
	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		SendGetData(payload.AddrFrom, "block", blockHash)
		blocksInTransit = blocksInTransit[1:]
	} else {
		// 如果下载完成, 重新索引UTXO集
		UTXOSet := UTXOSet{bc}
		UTXOSet.Reindex()
	}
}

// 处理接收到的inv信息
func handleInv(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)

	// 处理不同类型的inv信息
	// 请求"块"
	if payload.Type == "block" {
		// 记录待下载的块哈希列表
		blocksInTransit = payload.Items
		// 请求第一个块(在我们的实现中，我们永远也不会发送有多重哈希的 inv 消息，因此这里只会请求一个块)
		blockHash := payload.Items[0]
		// 给 inv 消息的发送者发送 getdata 命令并更新 blocksInTransit
		SendGetData(payload.AddrFrom, "block", blockHash)
		// 更新 blocksInTransit 列表，移除已请求的块哈希
		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	// 请求"交易"
	if payload.Type == "tx" {
		// 提取交易ID
		txID := payload.Items[0]
		// 如果内存池中没有该交易，则请求该交易数据
		if mempool[hex.EncodeToString(txID)].ID == nil {
			SendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

// 处理接收到的交易
func handleTx(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	// 获取交易数据并反序列化
	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	// 将交易添加到内存池, 以便后续打包进块
	mempool[hex.EncodeToString(tx.ID)] = tx

	// 如果当前节点是中心节点，不挖矿, 将新的交易推送给网络中的其他节点
	if nodeAddress == GetCentralNodeAddress() {
		for _, node := range knownNodes {
			if node != nodeAddress && node != payload.AddrFrom {
				SendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		// 如果当前节点（矿工）的内存池中有两笔或更多的交易，开始挖矿
		if len(mempool) >= 2 && len(miningAddress) > 0 {
		MineTransactions:
			var txs []*Transaction

			// 从内存池中选择有效交易
			for id := range mempool {
				tx := mempool[id]
				if bc.VerifyTransaction(&tx) {
					txs = append(txs, &tx)
				}
			}

			// 如果没有有效交易则退出
			if len(txs) == 0 {
				fmt.Println("All transactions are invalid! Waiting for new ones...")
				return
			}

			// 创建挖矿奖励交易
			cbTx := NewCoinbaseTX(miningAddress, "")
			txs = append(txs, cbTx)

			// 挖掘新块并将交易打包进块
			newBlock := bc.MineBlock(txs)
			// 更新UTXO集
			UTXOSet := UTXOSet{bc}
			UTXOSet.Reindex()

			fmt.Println("New block is mined!")

			// 从内存池中移除已打包进块的交易
			for _, tx := range txs {
				txID := hex.EncodeToString(tx.ID)
				delete(mempool, txID)
			}

			// 向其他节点广播新块
			for _, node := range knownNodes {
				if node != nodeAddress {
					SendInv(node, "block", [][]byte{newBlock.Hash})
				}
			}

			// 如果内存池中还有交易，继续挖矿
			if len(mempool) > 0 {
				goto MineTransactions
			}
		}
	}
}

// 编码数据为字节数组
// 空接口类型, 可以接收任意类型的数据
func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// 检查节点地址是否在已知节点列表中
func nodeIsKnown(addr string) bool {
	for _, node := range knownNodes {
		if node == addr {
			return true
		}
	}

	return false
}