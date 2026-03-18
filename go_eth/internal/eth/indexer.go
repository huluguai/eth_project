package eth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
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
	TokenAddress  string
	StartBlock    *uint64
	Confirmations uint64
	PollInterval  time.Duration
}

type Indexer struct {
	ethc *ethclient.Client
	db   *gorm.DB
	cfg  IndexerConfig

	tokenAddr common.Address
	stateKey  string
}

func NewIndexer(ethc *ethclient.Client, db *gorm.DB, cfg IndexerConfig) *Indexer {
	addr := common.HexToAddress(cfg.TokenAddress)
	return &Indexer{
		ethc:      ethc,
		db:        db,
		cfg:       cfg,
		tokenAddr: addr,
		// stateKey 用于把“索引进度”与具体 token 绑定，避免多 token/多实例共享同一进度造成跳块或重复扫。
		// 统一 lower-case 便于跨平台/跨调用方比较与排重。
		stateKey: "erc20_transfer:" + strings.ToLower(addr.Hex()),
	}
}

func (i *Indexer) Run(ctx context.Context) error {
	if i.cfg.PollInterval <= 0 {
		i.cfg.PollInterval = 8 * time.Second
	}
	if i.cfg.Confirmations == 0 {
		// Confirmations 用来规避链重组（reorg）。我们只处理“足够确认”的区块高度：latest-confirmations。
		i.cfg.Confirmations = 6
	}

	lastProcessed, err := i.loadLastProcessed(ctx)
	if err != nil {
		return err
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
			from := lastProcessed + 1
			for from <= target {
				to := from + maxRange - 1
				if to > target {
					to = target
				}

				if err := i.scanRange(ctx, from, to); err != nil {
					log.Printf("indexer: scan %d-%d error: %v", from, to, err)
					break
				}
				lastProcessed = to
				// 进度持久化放在每个成功段之后：即使进程重启，也能从最近成功的 to 继续，减少重复扫描成本。
				if err := i.saveLastProcessed(ctx, lastProcessed); err != nil {
					log.Printf("indexer: save state error: %v", err)
					break
				}

				from = to + 1
			}
		}
	}
}

func (i *Indexer) scanRange(ctx context.Context, from, to uint64) error {
	// ERC-20 Transfer 事件签名：Transfer(address,address,uint256)
	// 通过 Topics[0] 的 keccak256(signature) 过滤，只拉取 Transfer 相关日志。
	transferSigHash := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	q := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(to),
		Addresses: []common.Address{i.tokenAddr},
		Topics:    [][]common.Hash{{transferSigHash}},
	}
	logs, err := i.ethc.FilterLogs(ctx, q)
	if err != nil {
		return err
	}
	if len(logs) == 0 {
		return nil
	}

	transfers := make([]models.Transfer, 0, len(logs))
	for _, lg := range logs {
		t, err := decodeTransferLog(lg, i.tokenAddr.Hex())
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

func decodeTransferLog(lg types.Log, tokenAddr string) (models.Transfer, error) {
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

	out.TokenAddress = strings.ToLower(tokenAddr)
	out.TxHash = strings.ToLower(lg.TxHash.Hex())
	out.LogIndex = lg.Index
	out.BlockNumber = lg.BlockNumber
	out.FromAddress = strings.ToLower(from)
	out.ToAddress = strings.ToLower(to)
	out.Amount = amt.String()
	return out, nil
}

func (i *Indexer) loadLastProcessed(ctx context.Context) (uint64, error) {
	var st models.IndexerState
	err := i.db.WithContext(ctx).First(&st, "key = ?", i.stateKey).Error
	if err == nil {
		return st.LastProcessedBlock, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	// 首次启动没有状态时的策略：
	// - 若配置了 StartBlock：从指定高度开始（并立刻落库为初始进度）。
	// - 否则：默认从“最新确认高度”开始，避免回扫全链带来长时间冷启动。
	if i.cfg.StartBlock != nil {
		if err := i.saveLastProcessed(ctx, *i.cfg.StartBlock); err != nil {
			return 0, err
		}
		return *i.cfg.StartBlock, nil
	}

	latest, err := i.ethc.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	start := latest
	if latest > i.cfg.Confirmations {
		start = latest - i.cfg.Confirmations
	}
	if err := i.saveLastProcessed(ctx, start); err != nil {
		return 0, err
	}
	return start, nil
}

func (i *Indexer) saveLastProcessed(ctx context.Context, block uint64) error {
	st := models.IndexerState{
		Key:                i.stateKey,
		LastProcessedBlock: block,
	}
	// Upsert：保证并发启动/多实例时也能安全写入同一个 key 的进度。
	return i.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_processed_block", "updated_at"}),
	}).Create(&st).Error
}
