package blockchain

// import (
// 	"bytes"
// 	"testing"
// 	"github.com/stretchr/testify/assert"
// )

// // Test that a new blockchain contains exactly one genesis block with expected fields
// func TestNewBlockChain_Genesis(t *testing.T) {
// 	bc := NewBlockChain()
// 	if assert.NotNil(t, bc) {
// 		// blockchain should contain exactly one block (genesis)
// 		assert.Equal(t, 1, len(bc.block), "new blockchain should contain one genesis block")
// 		genesis := bc.block[0]
// 		assert.Equal(t, "Genesis Block", string(genesis.Data))
// 		// genesis prevhash should be empty
// 		assert.Equal(t, 0, len(genesis.PrevHash))
// 		// hash should be set
// 		assert.Greater(t, len(genesis.Hash), 0)
// 	}
// }

// // Test AddBlock links new blocks to the previous block's hash
// func TestAddBlock_Chaining(t *testing.T) {
// 	bc := NewBlockChain()
// 	bc.AddBlock("Send 1 BTC to Ivan")
// 	bc.AddBlock("Send 2 more BTC to Ivan")

// 	assert.Equal(t, 3, len(bc.block), "blockchain should have 3 blocks after adding two")

// 	// verify chaining: each block's PrevHash equals previous block's Hash
// 	for i := 1; i < len(bc.block); i++ {
// 		prev := bc.block[i-1]
// 		cur := bc.block[i]
// 		assert.True(t, bytes.Equal(prev.Hash, cur.PrevHash), "block %d PrevHash should equal hash of block %d", i, i-1)
// 	}
// }

// // Test SetHash produces deterministic hash for same content (timestamp included so we craft a block)
// func TestSetHash_Deterministic(t *testing.T) {
// 	// create two blocks with identical fields and same timestamp to ensure SetHash gives same result
// 	b1 := &Block{
// 		TimeStamp: 1234567890,
// 		Data:      []byte("hello"),
// 		PrevHash:  []byte("prev"),
// 	}
// 	b2 := &Block{
// 		TimeStamp: 1234567890,
// 		Data:      []byte("hello"),
// 		PrevHash:  []byte("prev"),
// 	}

// 	b1.SetHash()
// 	b2.SetHash()

// 	assert.Equal(t, b1.Hash, b2.Hash, "SetHash should produce same hash for identical block fields")
// }
