package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// 本文件实现了一个极简的工作量证明（Proof-of-Work, PoW）区块链示例。
// 注释旨在讲解区块链中常见概念：交易、区块、链、工作量证明、挖矿与链验证。

// Transation 表示一笔简单的交易：发送者、接收者、金额。
// 区块链中的交易通常记录在区块里，实际系统会包含签名、防篡改字段等。
// 这里为教学简化，未实现签名和账户校验。
// 注意拼写保持原样以与后续 JSON 标签一致。
type Transation struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int    `json:"amount"`
}

// Block 表示链上的一个区块。
// Index: 区块高度（从 1 开始）
// Timestamp: 区块被创建的 Unix 时间戳
// Transations: 本区块包含的交易列表
// Proof: 工作量证明值（一个整数，通常是通过大量计算找到的）
// PreviousHash: 上一个区块的哈希，用于将区块串联成链
type Block struct {
	Index        int          `json:"index"`
	Timestamp    int64        `json:"timestamp"`
	Transations  []Transation `json:"transactions"`
	Proof        int64        `json:"proof"`
	PreviousHash string       `json:"previous_hash"`
}

// Blockchain 表示一条链，包含已确认的区块链（Chain）和正在收集的交易池（CurrentTransations）。
// Difficulty 用于定义 PoW 算法的难度：要求哈希值以一定数量的前导零开头。
// 在真实链中，难度通常会根据出块时间动态调整。
type Blockchain struct {
	Chain              []Block
	CurrentTransations []Transation
	Difficulty         int
}

// NewBlockchain 初始化一个新的区块链实例。
// - 初始化链和难度
// - 创建创世区块（genesis block），这里将 proof 设置为 100，previousHash 设置为 "1"（任意值）
// 创世区块是链的第一个区块，通常由系统预设或在启动时创建。
func NewBlockchain() *Blockchain {
	bc := &Blockchain{
		Chain: []Block{},
		// 设置挖矿难度（前导零的数量）
		Difficulty: 4,
	}
	// 创建创世区块，保证链非空，便于后续引用 LastBlock()
	bc.NewBlock(100, "1")
	return bc
}

// NewBlock 创建一个新的区块并添加到链上。
// 参数：proof（找到的工作量证明），previousHash（上一个区块的哈希，若为空则从链中获取）
// 过程：
// - 根据传入或计算出的 previousHash 构造区块
// - 将当前交易池中的交易放入区块
// - 重置当前交易池
// - 将区块追加到链上并返回该区块指针
func (bc *Blockchain) NewBlock(proof int64, previousHash string) *Block {
	var hash string
	if previousHash == "" {
		// 如果未传入 previousHash，则计算链上最后一个区块的哈希
		hash = bc.Hash(bc.Chain[len(bc.Chain)-1])
	} else {
		hash = previousHash
	}
	block := Block{
		Index:        len(bc.Chain) + 1,
		Timestamp:    time.Now().Unix(),
		Transations:  bc.CurrentTransations,
		Proof:        proof,
		PreviousHash: hash,
	}
	// 新区块已被打包，清空当前交易池
	bc.CurrentTransations = []Transation{}
	bc.Chain = append(bc.Chain, block)
	return &block
}

// NewTransation 将一笔交易添加到当前交易池中，返回该交易将要被打包到的下一个区块索引。
// 实际系统中应有交易签名与验证逻辑，这里作简化演示。
func (bc *Blockchain) NewTransation(sender, recipient string, amount int) int {
	bc.CurrentTransations = append(bc.CurrentTransations, Transation{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	})
	// 返回交易将被打包进的区块索引（下一高度）
	return bc.LastBlock().Index + 1
}

// LastBlock 返回当前链上的最后一个区块的指针（最新区块）。
// 若链为空则返回 nil（但本实现通过创世区块保证链至少有一个区块）。
func (bc *Blockchain) LastBlock() *Block {
	if len(bc.Chain) == 0 {
		return nil
	}
	return &bc.Chain[len(bc.Chain)-1]
}

// Hash 计算区块的 SHA-256 哈希值。
// 说明：我们通过 JSON 序列化整个区块结构来获得一个字节序列，然后对其做哈希。
// 真实链中常采用更严格的序列化与字段顺序保证哈希的一致性与确定性。
func (bc *Blockchain) Hash(block Block) string {
	blockBytes, err := json.Marshal(block)
	if err != nil {
		return ""
	}
	hasher := sha256.New()
	hasher.Write(blockBytes)
	return hex.EncodeToString(hasher.Sum(nil))
}

// ValidProof 验证工作量证明的有效性（简单示例）。
// 本实现将上一个 proof 与当前 candidate proof 直接拼接为字符串并哈希，
// 然后检查结果哈希是否以指定数量的零开头（前导零策略）。
// 这是一种简单且直观的 PoW 验证方法，用于说明概念。
func (bc *Blockchain) ValidProof(lastProof int64, proof int64) bool {
	// 将上一个 proof 和当前 proof 连接形成待哈希字符串
	guess := fmt.Sprintf("%d%d", lastProof, proof)
	guessHash := sha256.Sum256([]byte(guess))
	hasStr := hex.EncodeToString(guessHash[:])
	// 验证哈希是否以指定数量的零开头（难度控制）
	return strings.HasPrefix(hasStr, strings.Repeat("0", bc.Difficulty))
}

// ProofOfWork 简单的穷举算法：不断尝试 proof 直到找到满足 ValidProof 的值。
// 在真实网络中，PoW 需要大量计算以保证攻击成本高昂，
// 这里为教学演示，不做任何优化。
func (bc *Blockchain) ProofOfWork(lastProof int64) int64 {
	var proof int64 = 0
	for !bc.ValidProof(lastProof, proof) {
		proof++
	}
	return proof
}

// ValidChain 验证给定链的有效性：
// - 每个区块的 previousHash 必须等于上一个区块计算得到的哈希
// - 每个区块包含的 proof 必须满足工作量证明的校验
// 这是区块链节点在接收到新链时用于验证其合法性的基本逻辑。
func (bc *Blockchain) ValidChain(chain []Block) bool {
	lastBlock := chain[0]
	currentIndex := 1

	for currentIndex < len(chain) {
		block := chain[currentIndex]
		// 验证前一区块的哈希是否正确
		if block.PreviousHash != bc.Hash(lastBlock) {
			return false
		}
		// 验证工作量证明
		if !bc.ValidProof(lastBlock.Proof, block.Proof) {
			return false
		}
		lastBlock = block
		currentIndex++
	}
	return true
}

// Mine 模拟矿工挖矿行为：
// 1. 获取最后一个区块的 proof，运行 PoW 算法找到新的 proof
// 2. 挖矿成功后，将一笔奖励交易（从地址 "0" 出账）加入交易池，奖励矿工
// 3. 创建新区块并将其追加到链上
// 注意：真实网络中，矿工会打包手续费更高或更多的交易，并广播新区块。
func (bc *Blockchain) Mine(minerAddress string) *Block {
	// 获取上一个区块及其 proof
	lastBlock := bc.LastBlock()
	lastProof := lastBlock.Proof
	// 运行工作量证明找到新的 proof
	proof := bc.ProofOfWork(lastProof)

	// 挖矿成功，奖励矿工 1 个单位（此处为演示）
	bc.NewTransation("0", minerAddress, 1)

	// 计算上一区块的哈希并创建新区块
	previousHash := bc.Hash(*lastBlock)
	block := bc.NewBlock(proof, previousHash)
	return block
}

func main() {
	// 演示：创建区块链、添加交易并挖矿
	blockChain := NewBlockchain()
	fmt.Println("挖矿中...")
	// 添加一些交易到交易池
	blockChain.NewTransation("Alice", "Bob", 5)
	blockChain.NewTransation("Bob", "Charlie", 2)
	// 挖矿（找到 proof 并将交易打包成区块）
	minedBlock := blockChain.Mine("miner_address")
	fmt.Printf("区块 %d 已被挖出\n", minedBlock.Index)
	fmt.Printf("Proof: %d \n", minedBlock.Proof)
	fmt.Printf("Previous Hash: %s \n", minedBlock.PreviousHash)

	// 再添加一些交易并进行第二次挖矿
	blockChain.NewTransation("Charlie", "Dave", 1)
	minedBlock2 := blockChain.Mine("miner_address_2")
	fmt.Printf("区块 %d 已被挖出\n", minedBlock2.Index)
	fmt.Printf("Proof: %d \n", minedBlock2.Proof)
	fmt.Printf("Previous Hash: %s \n", minedBlock2.PreviousHash)

	// 打印链的信息和内容
	fmt.Println("\n=== 区块链信息===")
	fmt.Printf("链长度: %d \n", len(blockChain.Chain))

	// 验证链的有效性
	if blockChain.ValidChain(blockChain.Chain) {
		fmt.Println("区块链有效")
	} else {
		fmt.Println("区块链无效")
	}

	// 打印每个区块的详细内容及其哈希，便于理解区块内部结构
	fmt.Println("\n=== 区块链内容 ===")
	for i, block := range blockChain.Chain {
		fmt.Printf("\n区块 #%d:\n", i+1)
		blockJSON, _ := json.MarshalIndent(block, "", "  ")
		fmt.Println(string(blockJSON))
		fmt.Printf("哈希值: %s\n", blockChain.Hash(block))
	}
}

// testPOW 演示如何测试 PoW 算法的速度与正确性
// - 计算从给定 lastProof 出发的 proof
// - 输出寻找 proof 所用时间和最终哈希校验结果
func testPOW() {
	bc := NewBlockchain()
	lastProof := int64(100)
	fmt.Printf("测试POW算法")
	strart := time.Now()
	proof := bc.ProofOfWork(lastProof)
	duration := time.Since(strart)
	fmt.Printf("找到的proof: %d, 耗时: %s\n", proof, duration)
	fmt.Printf("验证结果: %v\n", bc.ValidProof(lastProof, proof))

	// 手动计算并显示哈希，便于观察哈希值与难度的关系
	guess := fmt.Sprintf("%d%d", lastProof, proof)
	guessHash := sha256.Sum256([]byte(guess))
	hasStr := hex.EncodeToString(guessHash[:])
	fmt.Printf("计算的哈希值: %s\n", hasStr)
	fmt.Printf("哈希值是否满足条件: %v\n", strings.HasPrefix(hasStr, strings.Repeat("0", bc.Difficulty)))
}
