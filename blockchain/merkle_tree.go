package blockchain

import (
	"crypto/sha256"
) 

type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
    mNode := MerkleNode{}

	// 叶子节点
    if left == nil && right == nil {
        hash := sha256.Sum256(data)
        mNode.Data = hash[:]
    } else { // 非叶子节点
        prevHashes := append(left.Data, right.Data...)
        hash := sha256.Sum256(prevHashes)
        mNode.Data = hash[:]
    }

    mNode.Left = left
    mNode.Right = right

    return &mNode
}

func NewMerkleTree(data [][]byte) *MerkleTree {
    var nodes []MerkleNode

	// 确保data长度为偶数
	// 若长度为奇数，则复制最后一个元素补齐
    if len(data)%2 != 0 {
        data = append(data, data[len(data)-1])
    }

    for _, datum := range data {
		// 创建叶子节点
        node := NewMerkleNode(nil, nil, datum)
        nodes = append(nodes, *node)
    }

	// 外层循环：每轮处理当前层级，生成上一层节点，直到只剩根节点
    for i := 0; i < len(data)/2; i++ {
        var newLevel []MerkleNode

		// 内层循环：每次取两个相邻节点，生成它们的父节点
        for j := 0; j < len(nodes); j += 2 {
            node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
            newLevel = append(newLevel, *node)
        }

        nodes = newLevel // 当前层级节点替换为新生成的上一层节点
    }

    mTree := MerkleTree{&nodes[0]} // 通过根节点创建树

    return &mTree
}