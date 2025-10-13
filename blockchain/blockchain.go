package blockchain

import (
	"fmt"
	"github.com/boltdb/bolt"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"

type BlockChain struct{
	tip []byte 		// 用于存储区块链"末端"（最新区块）的哈希值
	db *bolt.DB		// 持久化存储区块链数据的数据库连接
}

type BlockChainIterator struct {
	currentHash []byte
	db 			*bolt.DB
}

func (bc *BlockChain) Iterator() *BlockChainIterator {
	return &BlockChainIterator{bc.tip, bc.db}
}

func (bc *BlockChain) AddBlock(data string) error {
	var lastHash []byte

	if err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", blocksBucket)
		}
		lastHash = b.Get([]byte("l"))
		return nil
	}); err != nil {
		return fmt.Errorf("failed to get last hash: %w", err)
	}

	newBlock := NewBlock(data, lastHash)

	if err := bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", blocksBucket)
		}
		if err := b.Put(newBlock.Hash, newBlock.Serialize()); err != nil {
			return err
		}
		if err := b.Put([]byte("l"), newBlock.Hash); err != nil {
			return err
		}
		bc.tip = newBlock.Hash
		return nil
	}); err != nil {
		return fmt.Errorf("failed to add new block: %w", err)
	}

	return nil
}

// 区块链中至少要有一个块，称为创世块
func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}

// 创建一个新的区块链,该链初始自带一个创世块
// 返回 (*BlockChain, error) 并将初始化过程中的错误向上返回
func NewBlockChain() (*BlockChain, error) {
	var tip []byte 
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	// bolt数据库的读写事务方法，用于执行修改数据库的操作
	err = db.Update(func (tx *bolt.Tx) error {
		 // 尝试从当前事务中获取名为blocksBucket的 "桶"，类似数据库的 "表"
		b := tx.Bucket([]byte(blocksBucket))

		if b == nil {
			genesis := NewGenesisBlock()
			b, err := tx.CreateBucket([]byte(blocksBucket))
			if err != nil {
				return err
			}
			if err = b.Put(genesis.Hash, genesis.Serialize()); err != nil {
				return err
			}
			// 用键"l"（"last"的缩写）记录最新区块的哈希（此时为创世块哈希）
			if err = b.Put([]byte("l"), genesis.Hash); err != nil {
				return err
			}
			tip = genesis.Hash // 更新tip为创世块哈希（链的末端是创世块）
		} else {
			// 若桶已存在，说明区块链已存在，从桶中读取最新区块的哈希
			tip = b.Get([]byte("l"))
		}
		return nil
	})

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize db: %w", err)
	}

	bc := BlockChain{tip, db}

	return &bc, nil
}

func (bc *BlockChain) CloseDB() {
	bc.db.Close()
}

// 从tip(末端)向前遍历区块链
// 同时杜绝了分支的问题(因为父Hash是唯一的)
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