package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"golang.org/x/crypto/ripemd160"
)

const version = byte(0x00)
const walletFile = "wallet.dat"
const addressChecksumLen = 4 // 地址校验和的长度（固定4字节，用于验证地址有效性）

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

func NewWallet() *Wallet {
	priKey, pubKey := newKeyPair()
	wallet := Wallet{priKey, pubKey}

	return &wallet
}

func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, _ := ecdsa.GenerateKey(curve, rand.Reader)
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubKey
}

func (w Wallet) GetAddress() []byte {
    // 步骤1：对公钥进行哈希处理，得到公钥哈希（RIPEMD160格式）
    pubKeyHash := HashPubKey(w.PublicKey)

    // 步骤2：拼接版本号和公钥哈希（用于标识地址版本）
    versionedPayload := append([]byte{version}, pubKeyHash...)

    // 步骤3：计算校验和（用于验证地址有效性）
    checksum := checksum(versionedPayload)

    // 步骤4：拼接版本化数据和校验和，得到完整数据
    fullPayload := append(versionedPayload, checksum...)

    // 步骤5：Base58编码（生成最终地址，便于人类识别和输入）
    address := Base58Encode(fullPayload)

    return address
}

func HashPubKey(pubKey []byte) []byte {
	// 第一步：对公钥做SHA-256哈希（压缩公钥长度，增强安全性）
    publicSHA256 := sha256.Sum256(pubKey)

    // 第二步：对SHA-256结果做RIPEMD160哈希（得到20字节的公钥哈希，比特币地址核心）
	RIPEMD160Hasher := ripemd160.New()
	RIPEMD160Hasher.Write(publicSHA256[:])
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}

// 当用户输入地址时，可通过重新计算校验和验证地址是否被篡改或输入错误
func checksum(payload []byte) []byte {
    // 第一步：对输入数据做SHA-256哈希
    firstSHA := sha256.Sum256(payload)
    // 第二步：对第一次哈希结果再做SHA-256哈希
    secondSHA := sha256.Sum256(firstSHA[:])
    // 取前4字节作为校验和
    return secondSHA[:addressChecksumLen]
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	fmt.Printf("totalKeyHash: %x\n", pubKeyHash)
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	fmt.Printf("actualChecksum: %x\n", actualChecksum)
	version := pubKeyHash[0]
	fmt.Printf("version: %x\n", version)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	fmt.Printf("pubKeyHash: %x\n", pubKeyHash)
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))
	fmt.Printf("targetChecksum: %x\n", targetChecksum)
	return bytes.Equal(actualChecksum, targetChecksum)
}
