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
		data = fmt.Sprintf("Reward to '%s'", to)
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

// 根据当前节点的input来更新spent, 遍历到前一个节点后根据当前节点的output来更新unspent,直到遍历的创世节点
func (bc *BlockChain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			// 将交易ID（字节数组）转换为十六进制字符串（方便作为map的key）
			txID := hex.EncodeToString(tx.ID)

		Outputs: // 标签，用于跳出当前循环到下一个输出
			// 遍历当前交易的所有输出（outIdx是输出索引，out是输出本身）
			for outIdx, out := range tx.Vout {
				// 检查当前输出是否已被花费
				// 末端节点一定没有被花费
				if spentTXOs[txID] != nil {
					// 遍历该交易中已被花费的输出索引
					for _, spentOutIdx := range spentTXOs[txID] {
						// 已被花费, 则跳过该输出
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}
				// 若输出未被花费，且可被目标地址解锁（即该输出属于目标地址）
				if out.IsLockedWithKey(pubKeyHash) {
					// 将当前交易加入未花费交易列表
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			if !tx.IsCoinbase() {
				// 遍历当前交易的所有输入
				for _, in := range tx.Vin {
					// 若输入可被目标地址解锁（即该输入是目标地址发起的，用于花费其之前的输出）
					if in.UsesKey(pubKeyHash) {
						// 输入来自的交易的ID,即输出交易的ID
						inTxID := hex.EncodeToString(in.Txid)
						// 将输出交易的ID中增加该输出的索引，表示该输出已被花费
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
					}
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTXs
}

// 将FindUnspentTransactions的结果进一步处理，得到所有未花费的输出slice
func (bc *BlockChain) FindUTXO(pubKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput
	unspentTransactions := bc.FindUnspentTransactions(pubKeyHash)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

func (bc *BlockChain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(pubKeyHash)
	accumulated := 0

Work:
	// 遍历该地址的所有未花费交易
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		// 遍历该交易的所有输出
		for outIdx, out := range tx.Vout {
			// 条件1：输出可被目标地址解锁（即该输出属于该地址）
			// 条件2：累计金额尚未达到目标金额（还需要继续筛选）
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				// 将该输出的金额加入累计
				accumulated += out.Value
				// 记录该输出的交易ID和索引，表示该输出可用于花费
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

				// 若累计金额已达到或超过目标金额，停止筛选
				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOutputs
}

func NewUTXOTransaction(from, to string, amount int, bc *BlockChain) (*Transaction, error) {
	var inputs []TXInput
	var outputs []TXOutput

	wallets, err := NewWallets()
	if err != nil {
		return nil, err
	}
	wallet := wallets.GetWallet(from)
	pubKeyHash := HashPubKey(wallet.PublicKey)
	acc, validOutputs := bc.FindSpendableOutputs(pubKeyHash, amount)
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
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc - amount, from))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	bc.SignTransaction(&tx, wallet.PrivateKey)

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
		// 计算txCopy的哈希（ID），此时的哈希包含了输入引用的前序输出信息、输出金额等核心内容
		txCopy.ID = txCopy.Hash()
		// 清空临时公钥
		txCopy.Vin[inID].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
		if err != nil {
			panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature
	}
}

func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

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

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}

		// 校验：使用公钥rawPubKey，验证签名(r,s)是否对应txCopy.ID（签名时的交易哈希）
		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}

	return true
}