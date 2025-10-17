package blockchain

import (
	"fmt"
	"github.com/boltdb/bolt"
)

type BlockChainIterator struct {
	currentHash []byte
	db 			*bolt.DB
}

// 在区块链数据库不存在时，创建一个全新的区块链数据库
// 生成创世块（区块链的第一个区块），并将创世块的挖矿奖励分配给指定地址
func (i *BlockChainIterator) Next() *Block {
	var block *Block

	if err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	}); err != nil {
		panic(fmt.Sprintf("failed to iterate to next block: %v", err))
	}

	i.currentHash = block.PrevHash

	return block
}

