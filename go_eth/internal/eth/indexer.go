// Package eth 实现区块链数据索引（例如 ERC-20 Transfer 日志）与持久化进度管理。
package eth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"
	"time"

	"eth_project/go_eth/internal/models"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IndexerConfig struct {
	TokenAddresses []string
	StartBlock    *uint64
	Confirmations uint64
	PollInterval  time.Duration
}

// Indexer 从区块链上持续抓取特定 token 的 Transfer 事件并落库。
type Indexer struct {
	ethc *ethclient.Client
	db   *gorm.DB
	cfg  IndexerConfig

	tokenAddrs []common.Address
	stateKey  string
}

func NewIndexer(ethc *ethclient.Client, db *gorm.DB, cfg IndexerConfig) *Indexer {
	tokenAddrs := make([]common.Address, 0, len(cfg.TokenAddresses))
	tokenHexes := make([]string, 0, len(cfg.TokenAddresses))
	for _, t := range cfg.TokenAddresses {
		// cfg 在 config 层已做 basic 校验（格式/长度），这里直接 hex-to-address。
		addr := common.HexToAddress(t)
		tokenAddrs = append(tokenAddrs, addr)
		tokenHexes = append(tokenHexes, strings.ToLower(addr.Hex()))
	}
	// 排序保证 token 顺序变化不会导致 stateKey 改变。
	sort.Strings(tokenHexes)
	stateHash := sha256.Sum256([]byte(strings.Join(tokenHexes, ",")))
	return &Indexer{
		ethc:      ethc,
		db:        db,
		cfg:       cfg,
		tokenAddrs: tokenAddrs,
		// stateKey 用于把“索引进度”与当前 token 集合绑定。
		stateKey: "erc20_transfer_multi:" + hex.EncodeToString(stateHash[:]),
	}
}

// Run 进入轮询模式：定期读取链上最新高度并处理“足够确认”的区块范围。
func (i *Indexer) Run(ctx context.Context) error {
	if i.cfg.PollInterval <= 0 {
		i.cfg.PollInterval = 8 * time.Second
	}
	if i.cfg.Confirmations == 0 {
		// Confirmations 用来规避链重组（reorg）。我们只处理“足够确认”的区块高度：latest-confirmations。
		i.cfg.Confirmations = 6
	}

	startMin := uint64(0)
	if i.cfg.StartBlock != nil {
		startMin = *i.cfg.StartBlock
	}
	reorgOverlap := i.cfg.Confirmations

	lastProcessed, storedBlockHash, err := i.loadLastProcessed(ctx)
	if err != nil {
		return err
	}

	// 若本地 checkpoint 的区块 hash 与链上 hash 不一致，说明上次索引点落在了重组区域。
	// 为避免“同高度不同区块”造成数据不一致，这里回退到重扫窗口。
	if storedBlockHash != "" {
		chainHash, err := i.blockHashAt(ctx, lastProcessed)
		if err != nil {
			log.Printf("indexer: check checkpoint hash failed: %v", err)
		} else if !strings.EqualFold(storedBlockHash, chainHash) {
			log.Printf("indexer: checkpoint reorg detected: block=%d localHash=%s chainHash=%s, backtrack=%d", lastProcessed, storedBlockHash, chainHash, reorgOverlap)
			if lastProcessed > reorgOverlap {
				lastProcessed = lastProcessed - reorgOverlap
			} else {
				lastProcessed = 0
			}
			if lastProcessed < startMin {
				lastProcessed = startMin
			}
		}
	}

	ticker := time.NewTicker(i.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			latest, err := i.ethc.BlockNumber(ctx)
			if err != nil {
				log.Printf("indexer: BlockNumber error: %v", err)
				continue
			}
			if latest <= i.cfg.Confirmations {
				continue
			}
			target := latest - i.cfg.Confirmations
			if target <= lastProcessed {
				continue
			}

			// 分段扫描的目的：控制单次 RPC 查询的区块跨度，避免 provider/节点对 FilterLogs 的限制导致超时或响应过大。
			// 这里即使某段失败，也不会推进 lastProcessed，保证不会“跳过”未成功入库的区块范围。
			const maxRange uint64 = 2000
			const minRange uint64 = 100
			rangeSize := maxRange

			// 重扫重叠窗口：在 lastProcessed 附近向前多扫一段，配合唯一约束确保幂等，
			// 以提高重组/确认不足时的数据一致性。
			from := lastProcessed + 1
			if from > reorgOverlap {
				from = from - reorgOverlap
			} else {
				from = 0
			}
			if from < startMin {
				from = startMin
			}
			for from <= target {
				to := from + rangeSize - 1
				if to > target {
					to = target
				}

				if err := i.scanRange(ctx, from, to); err != nil {
					log.Printf("indexer: scan %d-%d error: %v", from, to, err)
					// 自适应缩小区间：某些 provider 对过大的 FilterLogs 区间限制较严格。
					if rangeSize > minRange {
						rangeSize = rangeSize / 2
						if rangeSize < minRange {
							rangeSize = minRange
						}
						log.Printf("indexer: shrink scan range to %d (retry same from=%d)", rangeSize, from)
						continue
					}
					break
				}
				lastProcessed = to
				// 进度持久化放在每个成功段之后：即使进程重启，也能从最近成功的 to 继续，减少重复扫描成本。
				if err := i.saveLastProcessed(ctx, lastProcessed); err != nil {
					log.Printf("indexer: save state error: %v", err)
					break
				}

				from = to + 1
				// 成功后逐步放大区间，提升吞吐。
				if rangeSize < maxRange {
					rangeSize *= 2
					if rangeSize > maxRange {
						rangeSize = maxRange
					}
				}
			}
		}
	}
}

// scanRange 在 [from, to] 范围内抓取 Transfer 日志并写入数据库（保证幂等）。
func (i *Indexer) scanRange(ctx context.Context, from, to uint64) error {
	// ERC-20 Transfer 事件签名：Transfer(address,address,uint256)
	// 通过 Topics[0] 的 keccak256(signature) 过滤，只拉取 Transfer 相关日志。
	transferSigHash := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	q := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(to),
		Addresses: i.tokenAddrs,
		Topics:    [][]common.Hash{{transferSigHash}},
	}
	var logs []types.Log
	var err error
	// FilterLogs 失败时做指数退避重试，提升 provider 抖动/短暂错误下的稳定性。
	const maxAttempts = 4
	for attempt := 0; attempt < maxAttempts; attempt++ {
		logs, err = i.ethc.FilterLogs(ctx, q)
		if err == nil {
			break
		}
		if attempt == maxAttempts-1 {
			return err
		}
		backoff := time.Duration(250*(1<<attempt)) * time.Millisecond
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	if len(logs) == 0 {
		return nil
	}

	transfers := make([]models.Transfer, 0, len(logs))
	for _, lg := range logs {
		t, err := decodeTransferLog(lg)
		if err != nil {
			log.Printf("indexer: decode log error tx=%s idx=%d: %v", lg.TxHash.Hex(), lg.Index, err)
			continue
		}
		transfers = append(transfers, t)
	}
	if len(transfers) == 0 {
		return nil
	}

	// 同一笔交易在重扫/重启时可能再次写入；用唯一约束+DoNothing 保证幂等，不会把重复 transfer 插入为多条。
	return i.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&transfers).Error
}

func decodeTransferLog(lg types.Log) (models.Transfer, error) {
	var out models.Transfer
	if len(lg.Topics) < 3 {
		return out, errors.New("topics too short")
	}
	// indexed 参数编码在 topics 里：topics[1]=from, topics[2]=to。
	// 主题里是 32 字节右对齐的 address，需要截取后 20 字节。
	from := common.BytesToAddress(lg.Topics[1].Bytes()[12:]).Hex()
	to := common.BytesToAddress(lg.Topics[2].Bytes()[12:]).Hex()
	if len(lg.Data) != 32 {
		return out, fmt.Errorf("unexpected data len: %d", len(lg.Data))
	}
	// 非 indexed 的 value 位于 data，ERC-20 Transfer 的 uint256 正好是 32 字节。
	amt := new(big.Int).SetBytes(lg.Data)

	out.TokenAddress = strings.ToLower(lg.Address.Hex())
	out.TxHash = strings.ToLower(lg.TxHash.Hex())
	out.LogIndex = lg.Index
	out.BlockNumber = lg.BlockNumber
	out.FromAddress = strings.ToLower(from)
	out.ToAddress = strings.ToLower(to)
	out.Amount = amt.String()
	return out, nil
}

// loadLastProcessed 从数据库读取“最后处理到的区块高度”，并在首次启动时初始化起点。
func (i *Indexer) loadLastProcessed(ctx context.Context) (uint64, string, error) {
	var st models.IndexerState
	err := i.db.WithContext(ctx).First(&st, "key = ?", i.stateKey).Error
	if err == nil {
		return st.LastProcessedBlock, st.BlockHash, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, "", err
	}

	// 首次启动没有状态时的策略：
	// - 若配置了 StartBlock：从指定高度开始（并立刻落库为初始进度）。
	// - 否则：默认从“最新确认高度”开始，避免回扫全链带来长时间冷启动。
	if i.cfg.StartBlock != nil {
		if err := i.saveLastProcessed(ctx, *i.cfg.StartBlock); err != nil {
			return 0, "", err
		}
		// saveLastProcessed 会把 BlockHash 一并写入。
		return *i.cfg.StartBlock, "", nil
	}

	latest, err := i.ethc.BlockNumber(ctx)
	if err != nil {
		return 0, "", err
	}
	start := latest
	if latest > i.cfg.Confirmations {
		start = latest - i.cfg.Confirmations
	}
	if err := i.saveLastProcessed(ctx, start); err != nil {
		return 0, "", err
	}
	return start, "", nil
}

// blockHashAt 获取某个区块高度对应的区块头 hash（hex 字符串，统一 lower-case）。
func (i *Indexer) blockHashAt(ctx context.Context, block uint64) (string, error) {
	header, err := i.ethc.HeaderByNumber(ctx, new(big.Int).SetUint64(block))
	if err != nil {
		return "", err
	}
	return strings.ToLower(header.Hash().Hex()), nil
}

// saveLastProcessed 将当前处理进度写入数据库（对同一 key 使用 upsert，保证并发/多实例安全）。
func (i *Indexer) saveLastProcessed(ctx context.Context, block uint64) error {
	hash, err := i.blockHashAt(ctx, block)
	if err != nil {
		return err
	}
	st := models.IndexerState{
		Key:                i.stateKey,
		LastProcessedBlock: block,
		BlockHash:          hash,
	}
	// Upsert：保证并发启动/多实例时也能安全写入同一个 key 的进度。
	return i.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_processed_block", "block_hash", "updated_at"}),
	}).Create(&st).Error
}
