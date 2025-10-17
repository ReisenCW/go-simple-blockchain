package blockchain

import (
	"encoding/hex"
	"log"

	"github.com/boltdb/bolt"
)

const utxoBucket = "chainstate"// BoltDB中存储UTXO的桶名称(表)

// unspent transaction output set
type UTXOSet struct {
	Blockchain *BlockChain// 关联的区块链实例，用于获取全链数据
}

// 根据公钥哈希（对应一个地址）和目标金额，找到足够支付该金额的 UTXO，并返回累计金额和这些 UTXO 的位置（交易 ID + 输出索引）
func (u UTXOSet) FindSpendableOutputs(pubkeyHash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	accumulated := 0
	db := u.Blockchain.db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor() // 创建游标，用于遍历桶中所有数据

		for k, v := c.First(); k != nil; k, v = c.Next() {
			txID := hex.EncodeToString(k)// 交易ID转为字符串
			outs := DeserializeOutputs(v)// 反序列化，将字节流转为TXOutputs

			// 遍历当前交易的所有输出
			for outIdx, out := range outs.Outputs {
				// 检查输出是否属于该pubHash(地址)，且累计金额未达目标
				if out.IsLockedWithKey(pubkeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return accumulated, unspentOutputs
}

// 根据公钥哈希（地址），直接从 UTXO 集查询该地址所有的未花费交易输出（UTXO), 用于计算余额（余额 = 所有 UTXO 的 value 之和）
func (u UTXOSet) FindUTXO(pubKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput
	db := u.Blockchain.db

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			outs := DeserializeOutputs(v)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return UTXOs
}

// 统计 UTXO 集中包含多少笔交易（每笔交易可能有多个 UTXO）
func (u UTXOSet) CountTransactions() int {
	db := u.Blockchain.db
	counter := 0

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		// 遍历桶中所有键（每个键对应一笔交易），每有一个键则计数+1
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			counter++
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return counter
}

// 当 UTXO 集损坏或需要与区块链同步时，从区块链全量数据重建 UTXO 集
func (u UTXOSet) Reindex() {
	db := u.Blockchain.db
	bucketName := []byte(utxoBucket)

	// 第一步：删除旧的UTXO桶并创建新桶
	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != nil && err != bolt.ErrBucketNotFound {
			log.Panic(err)
		}

		_, err = tx.CreateBucket(bucketName)
		if err != nil {
			log.Panic(err)
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	// 第二步：从区块链中获取当前所有UTXO
	UTXO := u.Blockchain.FindUTXO()

	// 第三步：将UTXO写入新桶
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

		for txID, outs := range UTXO {
			key, err := hex.DecodeString(txID)
			if err != nil {
				log.Panic(err)
			}

			err = b.Put(key, outs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
}

// 当一个新块被添加到区块链时，更新 UTXO 集（移除被消耗的 UTXO，添加新产生的 UTXO）
func (u UTXOSet) Update(block *Block) {
	db := u.Blockchain.db

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))

		// 遍历区块中的所有交易
		for _, tx := range block.Transactions {
			// 处理非coinbase交易（coinbase交易没有输入，不需要消耗UTXO）
			if tx.IsCoinbase() == false {
				// 遍历交易的所有输入（Vin），这些输入引用了之前的UTXO，需要将其从UTXO集中移除
				for _, vin := range tx.Vin {
					updatedOuts := TXOutputs{} // 存储剩余的未花费输出
					// 根据输入引用的交易ID，从UTXO集中获取对应的输出列表
					outsBytes := b.Get(vin.Txid)
					outs := DeserializeOutputs(outsBytes)

					// 遍历该交易的所有输出，保留未被当前输入引用的输出
					for outIdx, out := range outs.Outputs {
						if outIdx != vin.Vout {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					// 如果剩余输出为空，则删除该交易的UTXO记录；否则更新记录
					if len(updatedOuts.Outputs) == 0 {
						err := b.Delete(vin.Txid)
						if err != nil {
							log.Panic(err)
						}
					} else {
						// 序列化剩余输出并更新到数据库
						err := b.Put(vin.Txid, updatedOuts.Serialize())
						if err != nil {
							log.Panic(err)
						}
					}

				}
			}

			// 处理当前交易的输出（Vout），将其作为新的UTXO加入集合
			newOutputs := TXOutputs{}
			for _, out := range tx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			// 将新输出序列化后存入UTXO集（键：当前交易ID；值：新输出列表）
			err := b.Put(tx.ID, newOutputs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}