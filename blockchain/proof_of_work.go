package blockchain

import (
	"bytes"
	"math/big"
	"fmt"
	"math"
	"crypto/sha256"
)

const targetBits = 24
const maxNonce = math.MaxInt64

func IntToHex(n int64) []byte {
	return []byte(fmt.Sprintf("%x", n))
}

type ProofOfWork struct {
	block *Block
	target *big.Int
}

func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	// 左移256 - targetBits位
	// 即 左侧开始数有targetBits个0
	target.Lsh(target, uint(256 - targetBits))
	return &ProofOfWork{block: b, target: target}
}

func (pow *ProofOfWork) PrepareData(nonce int) []byte {
	data := bytes.Join([][]byte{
		pow.block.PrevHash,
		pow.block.HashTransactions(),
		IntToHex(pow.block.TimeStamp),
		IntToHex(int64(nonce)),
	}, []byte{})
	return data
}

// 返回符合条件的nonce值和对应的hash
func (pow *ProofOfWork) Run() (int, []byte) {
	nonce := 0
	var hash [32]byte

	for nonce < maxNonce {
		data := pow.PrepareData(nonce)
		hash = sha256.Sum256(data)
		hashInt := new(big.Int).SetBytes(hash[:])
		// 比较hashInt和target的大小
		// 如果hashInt < target,则代表前面有targetBits个0,符合条件
		if hashInt.Cmp(pow.target) == -1 {
			break
		} else {
			nonce++
		}
	}
	return nonce, hash[:]
}

// 验证pow是否有效
func (pow *ProofOfWork) Validate() bool {
	data := pow.PrepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt := new(big.Int).SetBytes(hash[:])
	return hashInt.Cmp(pow.target) == -1
}