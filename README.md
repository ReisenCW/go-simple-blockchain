# 基于GO语言实现的简易区块链系统

一个使用 Go 语言从零实现的完整区块链系统，包含工作量证明（PoW）、UTXO 模型、数字签名、Merkle 树、简易多节点网络实现等核心功能。

## 项目环境

### 开发环境
- **Go 版本**: 1.24.9
- **操作系统**: Ubuntu
- **数据库**: BoltDB

### 主要依赖
- `github.com/boltdb/bolt` - 持久化存储区块链数据
- `golang.org/x/crypto` - 加密算法支持（RIPEMD160、ECDSA）
- `github.com/stretchr/testify` - 单元测试框架

### 安装与编译
```bash
# 克隆项目
git clone https://github.com/ReisenCW/go-simple-blockchain.git
cd go-simple-blockchain

# 下载依赖
go mod download

# 编译生成可执行文件
make
# 或手动编译
go build -o go-blockchain main.go
```

## 项目结构

```
go-simple-blockchain/
├── blockchain/           # 区块链核心模块
│   ├── block.go         # 区块结构定义
│   ├── blockchain.go    # 区块链管理
│   ├── blockchain_iterator.go  # 区块链迭代器
│   ├── proof_of_work.go # 工作量证明（PoW）
│   ├── transaction.go   # 交易结构与验证
│   ├── transaction_input.go    # 交易输入
│   ├── transaction_ouput.go    # 交易输出
│   ├── txo_set.go       # UTXO 集合管理
│   ├── merkle_tree.go   # Merkle 树实现
│   ├── wallet.go        # 钱包（密钥对管理）
│   ├── wallets.go       # 钱包集合管理
│   ├── base58.go        # Base58 编解码
│   ├── server.go        # P2P 网络节点
│   └── util.go          # 工具函数
├── cli/                 # 命令行接口
│   ├── cli.go           # CLI 主框架
│   └── cli_functions.go # CLI 命令实现
├── main.go              # 程序入口
├── go.mod               # Go 模块定义
├── Makefile             # 编译脚本
└── README.md            # 项目文档
```

## 主要模块

### 1. 区块（Block）
- **核心字段**:
  - `TimeStamp`: 区块创建时间戳
  - `Transactions`: 区块包含的交易列表
  - `PrevHash`: 前一个区块的哈希值（父哈希）
  - `Hash`: 当前区块的哈希值
  - `Nonce`: 工作量证明的随机数
  
- **功能**:
  - 使用 Merkle 树计算交易摘要
  - 通过 PoW 计算有效区块哈希
  - 序列化/反序列化支持持久化

### 2. 区块链（Blockchain）
- **数据结构**: 基于 BoltDB 的持久化链式存储
- **核心功能**:
  - 创建创世块（Genesis Block）
  - 添加新区块到链
  - 区块迭代与遍历
  - UTXO 集合管理（快速余额查询）
  - 交易签名与验证

### 3. 工作量证明（Proof of Work）
- **算法**: SHA-256 哈希计算
- **难度**: 目标值为 16 位前导零（可调节 `targetBits`）
- **流程**:
  1. 拼接区块数据（PrevHash + 交易哈希 + TimeStamp + Nonce）
  2. 计算 SHA-256 哈希
  3. 验证哈希是否小于目标值
  4. 若不满足则递增 Nonce 重新计算

### 4. UTXO 模型（Unspent Transaction Output）
- **设计理念**: 类似比特币的交易输出模型
- **核心机制**:
  - 每笔交易包含输入（TXInput）和输出（TXOutput）
  - 输入引用之前交易的未花费输出
  - 输出锁定到特定地址的公钥哈希
  - UTXO 集合缓存提升查询性能

### 5. 数字签名与验证
- **签名算法**: ECDSA（椭圆曲线数字签名）
- **曲线参数**: P-256（secp256r1）
- **流程**:
  - 发送方使用私钥对交易签名
  - 接收方使用发送方公钥验证签名
  - 防止交易篡改和双重支付

### 6. 钱包（Wallet）
- **密钥生成**: ECDSA 生成公私钥对
- **地址生成流程**:
  1. 对公钥进行 SHA-256 哈希
  2. 对结果进行 RIPEMD-160 哈希
  3. 添加版本前缀
  4. 计算校验和（双重 SHA-256 取前 4 字节）
  5. Base58 编码生成最终地址
  
- **持久化**: 使用 x509 DER 格式序列化私钥，避免 gob 序列化椭圆曲线内部结构

### 7. Merkle 树
- **用途**: 高效验证区块中的交易完整性
- **结构**: 二叉树，叶子节点为交易哈希，父节点为子节点哈希的组合哈希
- **优势**: 
  - 仅需 O(log n) 时间验证单笔交易
  - 支持简化支付验证（SPV）

### 8. 简易网络实现
- **节点类型**:
  - 中心节点（Central Node）: 初始区块链的引导节点
  - 钱包节点（Wallet Node）: 创建和发送交易
  - 矿工节点（Miner Node）: 挖矿打包交易
  
- **通信协议**:
  - `version`: 节点版本握手
  - `inv`: 通知可用区块/交易清单
  - `getdata`: 请求具体区块/交易数据
  - `block`: 传输区块数据
  - `tx`: 传输交易数据

## 功能实现

### CLI 命令列表

| 命令 | 参数 | 功能说明 |
|------|------|----------|
| `createwallet` | - | 生成新的钱包地址（ECDSA 密钥对） |
| `listaddresses` | - | 列出所有本地钱包地址 |
| `createblockchain` | `-address ADDRESS` | 创建新区块链并生成创世块，奖励发送至指定地址 |
| `getbalance` | `-address ADDRESS` | 查询指定地址的余额 |
| `send` | `-from FROM -to TO -amount AMOUNT [-mine]` | 发送交易，`-mine` 参数表示立即挖矿确认 |
| `printchain` | - | 打印区块链中的所有区块信息 |
| `reindexutxo` | - | 重建 UTXO 集合索引 |
| `startnode` | `[-miner ADDRESS]` | 启动 P2P 节点，`-miner` 参数指定挖矿奖励地址 |

### 使用示例

#### 1. 创建钱包
```bash
./go-blockchain createwallet
# 输出: Your new address: 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa
```

#### 2. 初始化区块链
```bash
./go-blockchain createblockchain -address 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa
# 输出: Done!
```

#### 3. 查询余额
```bash
./go-blockchain getbalance -address 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa
# 输出: Balance of '1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa': 10
```

#### 4. 发送交易
```bash
./go-blockchain send -from 1A1zP1eP... -to 1BvBMSEY... -amount 3 -mine
# 输出: Success!
```

#### 5. 启动节点（多节点测试）
```bash
# 终端 1 - 启动中心节点
export NODE_ID=3000
./go-blockchain startnode

# 终端 2 - 启动矿工节点
export NODE_ID=3001
./go-blockchain startnode -miner 1MinerAddress...
```

## 区块链节点交互测试详细说明
验证区块链多节点间的交易发送、区块同步、挖矿奖励及余额计算功能，模拟中心节点、钱包节点、矿工节点的协同工作流程。

### 前置准备
1. **环境要求**：  
   - 操作系统：Linux（本项目的开发环境以及以下命令示例基于此，Windows需将`export`替换为`set`）  
   - Go环境：已配置Go编译环境  
   - 终端：至少3个（分别对应3个节点）  

2. **程序编译**：  
在项目根目录执行make命令或手动编译，生成可执行文件：
    ```bash
    # 编译生成可执行文件
    make
    # 或手动编译
    go build -o go-blockchain main.go
    ```

### 测试步骤

#### 阶段1：初始化中心节点（NODE 3000）
**作用**：创建创世块、作为初始区块链节点，负责向其他钱包地址转账。

1. **打开终端1，设置环境变量NODE_ID作为节点ID**：  
   ```bash
   export NODE_ID=3000  # 声明当前节点标识为3000
   ```

2. **创建中心节点钱包（CENTRAL_NODE）**：  
   执行命令生成中心节点地址，记录该地址（后续用`CENTRAL_NODE`指代）：  
   ```bash
   ./blockchain_go createwallet
   # 输出示例：Your new address: 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa（假地址，以实际输出为准）
   ```

3. **创建包含创世块的区块链**：  
   以中心节点地址作为创世块奖励接收者，初始化区块链：  
   ```bash
   ./blockchain_go createblockchain -address CENTRAL_NODE
   # 输出示例：Done!（此时生成区块链数据库文件blockchain_3000.db）
   ```

4. **备份创世块数据库**：  
   创世块是区块链的唯一标识，需复制给其他节点作为初始化依据：  
   ```bash
   cp blockchain_3000.db blockchain_genesis.db  # 备份创世块到通用文件
   ```

5. **向钱包节点转账并挖矿**：  
   向后续钱包节点的地址转账（暂时记录命令，后续在钱包节点生成地址后执行）：  
   ```bash
   # 后续步骤中，替换WALLET_1和WALLET_2为实际地址
   ./blockchain_go send -from CENTRAL_NODE -to WALLET_1 -amount 10 -mine
   ./blockchain_go send -from CENTRAL_NODE -to WALLET_2 -amount 10 -mine
   # 说明：-mine参数表示当前节点立即挖矿确认交易，生成新块
   ```

6. **启动中心节点服务**：  
   启动节点监听网络连接，用于与其他节点同步区块：  
   ```bash
   ./blockchain_go startnode
   # 输出示例：Starting node 3000（节点将持续运行，请勿关闭终端）
   ```


#### 阶段2：初始化钱包节点（NODE 3001）
**作用**：生成用户钱包地址，接收中心节点转账，发起新交易。

1. **打开终端2，设置节点ID**：  
   ```bash
   export NODE_ID=3001  # 声明当前节点标识为3001
   ```

2. **生成钱包地址**：  
   执行3次`createwallet`命令，生成3个地址，分别记录为`WALLET_1`、`WALLET_2`、`WALLET_3`：  
   ```bash
   ./blockchain_go createwallet  # 生成WALLET_1
   ./blockchain_go createwallet  # 生成WALLET_2
   ./blockchain_go createwallet  # 生成WALLET_3
   # 输出示例：每次执行会返回一个新地址，需记录下来
   ```

3. **初始化区块链（基于创世块）**：  
   复制中心节点备份的创世块数据库，作为本节点区块链的起点：  
   ```bash
   cp blockchain_genesis.db blockchain_3001.db  # 从创世块初始化本节点数据库
   ```

4. **启动钱包节点服务（同步区块）**：  
   启动节点，自动连接中心节点并同步区块：  
   ```bash
   ./blockchain_go startnode
   # 输出示例：
   # Starting node 3001
   # Received version command
   # （节点会从中心节点同步之前的转账区块，运行片刻后按Ctrl+C暂停）
   ```

5. **验证余额（同步后）**：  
   暂停节点后，检查各地址余额是否正确：  
   ```bash
   # 检查WALLET_1余额（预期10）
   ./blockchain_go getbalance -address WALLET_1
   # 检查WALLET_2余额（预期10）
   ./blockchain_go getbalance -address WALLET_2
   # 检查中心节点余额（创世块奖励+剩余，预期初始奖励减去已转账部分，示例10）
   ./blockchain_go getbalance -address CENTRAL_NODE
   ```


#### 阶段3：初始化矿工节点（NODE 3002）
**作用**：负责挖矿确认交易，获取挖矿奖励。

1. **打开终端3，设置节点ID**：  
   ```bash
   export NODE_ID=3002  # 声明当前节点标识为3002
   ```

2. **生成矿工钱包地址**：  
   执行`createwallet`生成矿工地址，记录为`MINER_WALLET`：  
   ```bash
   ./blockchain_go createwallet
   # 输出示例：Your new address: 1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2（假地址，以实际输出为准）
   ```

3. **初始化区块链（基于创世块）**：  
   复制创世块数据库，与其他节点保持区块链一致性：  
   ```bash
   cp blockchain_genesis.db blockchain_3002.db
   ```

4. **启动矿工节点（开启挖矿模式）**：  
   启动节点并指定矿工地址（挖矿奖励将转入该地址）：  
   ```bash
   ./blockchain_go startnode -miner MINER_WALLET
   # 输出示例：
   # Starting node 3002
   # Mining is on. Address to receive rewards: MINER_WALLET
   # （节点将持续运行，等待接收交易并挖矿）
   ```


#### 阶段4：发送交易并验证挖矿与同步
1. **钱包节点（NODE 3001）发送交易**：  
   在终端2中重新启动钱包节点（若已暂停），执行交易命令：  
   ```bash
   # 重新启动钱包节点（确保与其他节点连接）
   ./blockchain_go startnode
   # 按Ctrl+C暂停，执行交易（无需-mine，由矿工节点挖矿）
   ./blockchain_go send -from WALLET_1 -to WALLET_3 -amount 1
   ./blockchain_go send -from WALLET_2 -to WALLET_4 -amount 1  # WALLET_4可提前生成
   ```

2. **矿工节点（NODE 3002）挖矿确认**：  
   观察终端3，矿工节点会收到交易并挖矿，输出类似：  
   ```
   Recevied a new block!
   Added block [哈希值]
   ```

3. **钱包节点（NODE 3001）同步区块**：  
   在终端2中重启钱包节点，自动同步矿工节点挖出的新块：  
   ```bash
   ./blockchain_go startnode
   # 输出示例：Recevied a new block!（运行片刻后按Ctrl+C暂停）
   ```

4. **验证最终余额**：  
   在终端2中执行余额检查命令，预期结果如下：  
   ```bash
   # WALLET_1：10 - 1 = 9
   ./blockchain_go getbalance -address WALLET_1
   # WALLET_2：10 - 1 = 9
   ./blockchain_go getbalance -address WALLET_2
   # WALLET_3：0 + 1 = 1
   ./blockchain_go getbalance -address WALLET_3
   # WALLET_4：0 + 1 = 1
   ./blockchain_go getbalance -address WALLET_4
   # MINER_WALLET：获得挖矿奖励（示例10）
   ./blockchain_go getbalance -address MINER_WALLET
   ```


### 预期结果总结
| 地址         | 最终余额（示例） | 说明                 |
| ------------ | ---------------- | -------------------- |
| WALLET_1     | 9                | 转出1个单位后剩余    |
| WALLET_2     | 9                | 转出1个单位后剩余    |
| WALLET_3     | 1                | 接收WALLET_1转入     |
| WALLET_4     | 1                | 接收WALLET_2转入     |
| MINER_WALLET | 10               | 挖矿奖励（区块奖励） |


### 注意事项
1. **地址有效性**：确保所有地址通过`createwallet`生成
2. **节点连接**：若节点无法同步，检查`knownNodes`配置（默认包含`localhost:3000`），确保节点端口正确。  
3. **数据库文件**：每个节点的数据库文件（`blockchain_XXX.db`）需独立，避免互相覆盖。  
4. **挖矿确认**：交易需等待矿工节点挖矿生成新块后才会生效，若长时间未确认，检查矿工节点是否正常运行。
