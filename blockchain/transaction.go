package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
)

const subsidy = 10 // 挖矿奖励

// 由交易, 输入 和 输出 组成
type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	encoder := gob.NewEncoder(&encoded)
	err := encoder.Encode(tx)
	if err != nil {
		panic(err)
	}
	hash := sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

func (tx *Transaction) PrintTransaction() {
	fmt.Printf("Transaction ID: %x\n", tx.ID)
	for i, input := range tx.Vin {
		fmt.Printf(" Input %d:\n", i)
		fmt.Printf("  Txid: %x\n", input.Txid)
		fmt.Printf("  Vout: %d\n", input.Vout)
		fmt.Printf("  ScriptSig: %s\n", input.ScriptSig)
	}
}

// TXInput 包含 3 部分
// Txid: 一个交易输入引用了之前一笔交易的一个输出, ID 表明是之前哪笔交易
// Vout: 一笔交易可能有多个输出，Vout 为输出的索引
// ScriptSig: 提供解锁输出 Txid:Vout 的数据
type TXInput struct {
	Txid      []byte
	Vout      int
	ScriptSig string
}

// TXOutput 包含两部分
// Value: 有多少币，就是存储在 Value 里面
// ScriptPubKey: 对输出进行锁定
// 在当前实现中，ScriptPubKey 将仅用一个字符串来代替
type TXOutput struct {
	Value        int
	ScriptPubKey string
}

// 创建创世块时最早的交易(输出)
func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Reward to '%s'", to)
	}

	txin := TXInput{[]byte{}, -1, data} // 没有输入
	txout := TXOutput{subsidy, to}
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{txout}}
	tx.SetID()

	return &tx
}

func (in *TXInput) CanUnlockOutputWith(unlockingData string) bool {
	return in.ScriptSig == unlockingData
}

func (out *TXOutput) CanBeUnlockedWith(unlockingData string) bool {
	return out.ScriptPubKey == unlockingData
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// 根据当前节点的input来更新spent, 遍历到前一个节点后根据当前节点的output来更新unspent,直到遍历的创世节点
func (bc *BlockChain) FindUnspentTransactions(address string) []Transaction {
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
					for _, spentOut := range spentTXOs[txID] {
						// 已被花费, 则跳过该输出
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				// 若输出未被花费，且可被目标地址解锁（即该输出属于目标地址）
				if out.CanBeUnlockedWith(address) {
					// 将当前交易加入未花费交易列表
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			if !tx.IsCoinbase() {
				// 遍历当前交易的所有输入
				for _, in := range tx.Vin {
					// 若输入可被目标地址解锁（即该输入是目标地址发起的，用于花费其之前的输出）
					if in.CanUnlockOutputWith(address) {
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
func (bc *BlockChain) FindUTXO(address string) []TXOutput {
	var UTXOs []TXOutput
	unspentTransactions := bc.FindUnspentTransactions(address)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if out.CanBeUnlockedWith(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

func (bc *BlockChain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(address)
	accumulated := 0

Work:
	// 遍历该地址的所有未花费交易
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		// 遍历该交易的所有输出
		for outIdx, out := range tx.Vout {
			// 条件1：输出可被目标地址解锁（即该输出属于该地址）
			// 条件2：累计金额尚未达到目标金额（还需要继续筛选）
			if out.CanBeUnlockedWith(address) && accumulated < amount {
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

	acc, validOutputs := bc.FindSpendableOutputs(from, amount)

	if acc < amount {
		return nil, fmt.Errorf("ERROR: Not enough funds")
	}

	// Build a list of inputs
	for txid, outs := range validOutputs {
		txID, _ := hex.DecodeString(txid)

		for _, out := range outs {
			input := TXInput{txID, out, from}
			inputs = append(inputs, input)
		}
	}

	// 输出到接收方
	outputs = append(outputs, TXOutput{amount, to})
	// 找零
	if acc > amount {
		outputs = append(outputs, TXOutput{acc - amount, from})
	}

	tx := Transaction{nil, inputs, outputs}
	tx.SetID()

	return &tx, nil
}
