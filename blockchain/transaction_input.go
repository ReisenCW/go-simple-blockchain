package blockchain

import "bytes"

// TXInput 包含 4 部分
// Txid: 一个交易输入引用了之前一笔交易的一个输出, ID 表明是之前哪笔交易
// Vout: 一笔交易可能有多个输出，Vout 为输出的索引
// Signature: 提供解锁输出 Txid:Vout 的数据
// PubKey: 交易输入的公钥
type TXInput struct {
	Txid      []byte
	Vout      int
	Signature []byte
	PubKey    []byte
}

// 验证输入是否属于某个公钥哈希
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}