package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/boltdb/bolt"
)

const dbFile = "blockchain.db"
const blocksBucket = "blocks"
const genesisCoinbaseData = "The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"

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

func (bc *BlockChain) MineBlock(transactions []*Transaction) error {
	var lastHash []byte

	for _, tx := range transactions {
		if !bc.VerifyTransaction(tx) {
			panic("ERROR: Invalid transaction")
		}
	}

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

	newBlock := NewBlock(transactions, lastHash)

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
func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{})
}

func IsDataBaseExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// 当区块链数据库已存在（包含创世块及后续区块）时
// 通过该函数连接到现有数据库，初始化区块链实例，获取链的最新状态（末端哈希），供后续操作（如添加区块、查询交易等）使用
func NewBlockChain(address string) (*BlockChain, error) {
	if !IsDataBaseExists() {
		fmt.Println("No existing blockchain found. Create one first.")
		os.Exit(1)
	}
	var tip []byte 
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	// bolt数据库的读写事务方法，用于执行修改数据库的操作
	err = db.Update(func (tx *bolt.Tx) error {
		// 尝试从当前事务中获取名为blocksBucket的 "桶"，类似数据库的 "表"
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))
		return nil
	})

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize db: %w", err)
	}

	bc := BlockChain{tip, db}

	return &bc, nil
}

// CreateBlockchain 创建一个新的区块链数据库
// address 用来接收挖出创世块的奖励
func CreateBlockChain(address string) (*BlockChain, error) {
	if IsDataBaseExists() {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}
	var tip []byte 
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	// bolt数据库的读写事务方法，用于执行修改数据库的操作
	err = db.Update(func (tx *bolt.Tx) error {
		// 创建创世块
		cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
		genesis := NewGenesisBlock(cbtx)

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

func (bc *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction is not found")
}

func (bc *BlockChain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, _ := bc.FindTransaction(vin.Txid)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

func (bc *BlockChain) VerifyTransaction(tx *Transaction) bool {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, _ := bc.FindTransaction(vin.Txid)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}