package blockchain

import "bytes"

// TXOutput 包含两部分
// Value: 有多少币，就是存储在 Value 里面
// PubKeyHash: 锁定该输出的公钥哈希（只有拥有对应私钥的人才能解锁）
type TXOutput struct {
	Value        int
	PubKeyHash   []byte
}

// 创建输出时，将其 “绑定” 到一个地址（即只有该地址的所有者才能花费）
func (out *TXOutput) Lock(address []byte) {
    pubKeyHash := Base58Decode(address)  // 对地址进行Base58解码，得到原始字节（包含版本号、公钥哈希、校验和）
    pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]  // 截取公钥哈希：去掉第1个字节（版本号）和最后4个字节（校验和）
    out.PubKeyHash = pubKeyHash  // 将公钥哈希赋值给输出，完成锁定
}

// 验证输出是否被某个公钥哈希锁定（即，该输出是否属于目标地址的所有者）
func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

// 创建一个新的TXOutput
func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{value, nil}
	txo.Lock([]byte(address))

	return txo
}