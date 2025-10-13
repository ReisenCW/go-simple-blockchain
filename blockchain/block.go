package blockchain

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"
)

// Timestamp		当前时间戳，也就是区块创建的时间
// Data				区块存储的实际有效信息，也就是交易信息
// Hash				当前块的哈希
// PrevHash			前一个块的哈希，即父哈希
type Block struct {
	TimeStamp int64
	Data      []byte
	Hash      []byte
	PrevHash  []byte
}

func (b *Block) PrintBlock() {
	fmt.Printf("================\nBlock %x:\nPrevHash: %x\nData: %s\nTimeStamp: %d\n", b.Hash, b.PrevHash, b.Data, b.TimeStamp)
}

func (b *Block) SetHash() {
	// int转字符串, 10:十进制
	timestamp := []byte(strconv.FormatInt(b.TimeStamp, 10))
	// 直接拼接
	headers := bytes.Join([][]byte{b.Data, timestamp, b.PrevHash}, []byte{})
	hash := sha256.Sum256(headers)
	b.Hash = hash[:]
}

func NewBlock(data string, prevHash []byte) *Block {
	block := &Block {
		TimeStamp: time.Now().Unix(),
		Data:      []byte(data),
		PrevHash:  prevHash,
		Hash:      []byte{},
	}
	block.SetHash()
	return block
}
