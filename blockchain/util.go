package blockchain

import "fmt"

// 将整数转换为十六进制字节数组
func IntToHex(n int64) []byte {
	return []byte(fmt.Sprintf("%x", n))
}

// 把字节数组反转
func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}