package blockchain

import (
	"bytes"
	"math/big"
)

// 没有0,O,l,I这几个容易混淆的字符, 以及 + 和 / 两个在URL中有特殊含义的字符
// 每个字符在切片中的索引（0-57）对应其 “数值”（类似十进制中 0-9 的数值），编码和解码都依赖这个索引映射。
var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

// 将原始字节数组转换为 Base58 编码的字节数组（最终钱包地址的字符形式）
// 核心逻辑：将字节数组转换为 58 进制，再映射到b58Alphabet的字符，同时保留原始数据的前导零
func Base58Encode(input []byte) []byte {
	var result []byte

	x := big.NewInt(0).SetBytes(input) // 将字节数组转换为大整数

	base := big.NewInt(int64(len(b58Alphabet))) // 基数(58)
	zero := big.NewInt(0)
	mod := &big.Int{} // 用于存储除法的余数

	// 当x不等于0时，持续除以58取余数
	for x.Cmp(zero) != 0 { 
		// x = x / 58，mod = x % 58（余数）
		x.DivMod(x, base, mod)
		// 余数作为索引，取对应字符追加到result
		result = append(result, b58Alphabet[mod.Int64()])
	}

	// 将result反转，得到正确的58进制字符顺序
	ReverseBytes(result)

	// 原始字节中的0x00是有意义的（比如地址中的版本号前可能有零），但在之前的除法中，前导零会被忽略
	// 因此这里需要补回这些前导零
	for _, b := range input {  // 遍历原始输入字节
		if b == 0x00 {  // 若原始字节是0x00（前导零）
			result = append([]byte{b58Alphabet[0]}, result...)  // 在结果前加一个b58Alphabet[0]（即'1'）
		} else {
			break  // 遇到非0字节则停止（只处理连续前导零）
		}
	}

	return result
}

// 将 Base58 编码的字节数组解码为原始字节数组
// 核心逻辑：将 Base58 字符映射回其数值，转换为大整数，再转为字节数组，同时保留前导零
func Base58Decode(input []byte) []byte {
	result := big.NewInt(0)  // 用于存储解码后的大整数
	zeroBytes := 0  // 记录前导'1'的数量（对应原始的0x00）

	for _, b := range input {
		if b == b58Alphabet[0] {
			zeroBytes++
		}
	}
	// 去掉前导'1'后的部分（实际参与58进制计算的字符）
	payload := input[zeroBytes:]
	for _, b := range payload {
		// 获取b对应的index
		charIndex := bytes.IndexByte(b58Alphabet, b)
		// result = result * 58 + charIndex
		result.Mul(result, big.NewInt(58))
		result.Add(result, big.NewInt(int64(charIndex)))
	}

	decoded := result.Bytes()
	// 在前面追加zeroBytes个0x00（前导零）
	// decoded... 会将 decoded 切片 “展开” 为一系列独立的 byte 元素（比如 decoded = []byte{0x01, 0x02} 会被展开为 0x01, 0x02），这样 append 才能正确地将这些元素逐个追加到前缀切片后面
	decoded = append(bytes.Repeat([]byte{byte(0x00)}, zeroBytes), decoded...)

	return decoded
}

