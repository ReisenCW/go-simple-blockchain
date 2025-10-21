package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
)

const subsidy = 10 // 挖矿奖励

// 由交易, 输入 和 输出 组成
type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer
	encoder := gob.NewEncoder(&encoded)
	err := encoder.Encode(tx)
	if err != nil {
		panic(err)
	}
	return encoded.Bytes()
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// 创建创世块时最早的交易(输出)
func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		randData := make([]byte, 20)
		_, err := rand.Read(randData)
		if err != nil {
			panic(err)
		}

		data = fmt.Sprintf("%x", randData)
	}

	txin := TXInput{[]byte{}, -1, nil, []byte(data)} // 没有输入
	txout := NewTXOutput(subsidy, to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}}
	tx.ID = tx.Hash()

	return &tx
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))

	for i, input := range tx.Vin {

		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

func NewUTXOTransaction(wallet *Wallet, to string, amount int, UTXOSet *UTXOSet) (*Transaction, error) {
	var inputs []TXInput
	var outputs []TXOutput

	pubKeyHash := HashPubKey(wallet.PublicKey)
	acc, validOutputs := UTXOSet.FindSpendableOutputs(pubKeyHash, amount)
	if acc < amount {
		return nil, fmt.Errorf("ERROR: Not enough funds")
	}

	// Build a list of inputs
	for txid, outs := range validOutputs {
		txID, _ := hex.DecodeString(txid)

		for _, out := range outs {
			input := TXInput{txID, out, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}

	// 输出到接收方
	outputs = append(outputs, *NewTXOutput(amount, to))
	// 找零
	from := fmt.Sprintf("%s", wallet.GetAddress())
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc - amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	UTXOSet.Blockchain.SignTransaction(&tx, wallet.PrivateKey)

	return &tx, nil
}


// 生成当前交易的精简副本（Trimmed Copy），用于后续的签名过程。
// 签名需要基于交易的核心信息（如输入引用的前序交易、输出金额等），但不需要包含现有签名或公钥（这些是待生成或临时的信息）。
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, nil, nil})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{vout.Value, vout.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	txCopy := tx.TrimmedCopy()

	// 交易的每个输入可能来自不同的前序交易，因此需要逐个处理
	for inID, vin := range txCopy.Vin {
		// 找到对应的前序交易
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		// 临时填充公钥哈希：将txCopy当前输入的PubKey设为前序交易对应输出的PubKeyHash（前序交易的接收者的公钥哈希，即当前交易发起者的 “地址”）。
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash

		// 重新计算交易ID（哈希）
		dataToSign := fmt.Sprintf("%x\n", txCopy)
		r, s, err := ecdsa.Sign(rand.Reader, &privKey, []byte(dataToSign))
		if err != nil {
			panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature
		txCopy.Vin[inID].PubKey = nil
	}
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		dataToVerify := fmt.Sprintf("%x\n", txCopy)
		rawPubKey := ecdsa.PublicKey{curve, &x, &y}

		// 校验：使用公钥rawPubKey，验证签名(r,s)是否对应txCopy.ID（签名时的交易哈希）
		if ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) == false {
			return false
		}
		txCopy.Vin[inID].PubKey = nil
	}

	return true
}

func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	if err != nil {
		log.Panic(err)
	}

	return transaction
}