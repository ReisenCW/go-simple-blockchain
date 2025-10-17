package blockchain

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
)

// Timestamp		当前时间戳，也就是区块创建的时间
// Data				区块存储的实际有效信息，也就是交易信息
// Hash				当前块的哈希
// PrevHash			前一个块的哈希，即父哈希
type Block struct {
	TimeStamp   	int64
	Transactions 	[]*Transaction
	Hash         	[]byte
	PrevHash    	[]byte
	Nonce       	int
}

func (b *Block) PrintBlock() {
	fmt.Printf("================\nBlock %x:\nPrevHash: %x\n", b.Hash, b.PrevHash)
	for _, tx := range b.Transactions {
		fmt.Println(tx.String())
	}
	fmt.Printf("Timestamp: %d\n", b.TimeStamp)
	fmt.Printf("Nonce: %d\n", b.Nonce)
}

func NewBlock(transactions []*Transaction, prevHash []byte) *Block {
	block := &Block {
		TimeStamp: 			time.Now().Unix(),
		Transactions:      	transactions,
		PrevHash:  			prevHash,
		Hash:      			[]byte{},
	}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	block.Hash, block.Nonce = hash[:], nonce
	return block
}


// 在 BoltDB 中，值只能是 []byte 类型, 因此需要序列化和反序列化
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	// 创建一个将数据写入result缓冲区的encoder
	encoder := gob.NewEncoder(&result)
	encoder.Encode(b)
	return result.Bytes()
}

func DeserializeBlock(d []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(d))
	decoder.Decode(&block)
	return &block
}

func (block *Block) HashTransactions() []byte {
	var transactions [][]byte

	for _, tx := range block.Transactions {
		transactions = append(transactions, tx.ID)
	}
	mTree := NewMerkleTree(transactions)
	return mTree.RootNode.Data
}