package blockchain

type BlockChain struct{
	block []*Block
}

func (bc *BlockChain) AddBlock(data string) {
	previousBlock := bc.block[len(bc.block) - 1]
	newBlock := NewBlock(data, previousBlock.Hash)
	bc.block = append(bc.block, newBlock)
}

// 区块链中至少要有一个块，称为创世块
func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}

// 创建一个新的区块链,该链初始自带一个创世块
func NewBlockChain() *BlockChain {
	return &BlockChain{[]*Block{NewGenesisBlock()}}
}