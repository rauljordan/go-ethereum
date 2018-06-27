// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/database"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/txpool"
	"github.com/ethereum/go-ethereum/sharding/utils"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	config       *params.Config
	client       *mainchain.SMCClient
	p2p          *p2p.Server
	txpool       *txpool.TXPool
	txpoolSub    event.Subscription
	dbService    *database.ShardDB
	shardID      int
	shard        *sharding.Shard
	ctx          context.Context
	cancel       context.CancelFunc
	errChan      chan error
	requestsChan chan *types.Transaction
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a p2p network,
// and a shard transaction pool.
func NewProposer(config *params.Config, client *mainchain.SMCClient, p2p *p2p.Server, txpool *txpool.TXPool, dbService *database.ShardDB, shardID int) (*Proposer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	return &Proposer{
		config,
		client,
		p2p,
		txpool,
		nil,
		dbService,
		shardID,
		nil,
		ctx,
		cancel,
		errChan,
		nil,
	}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() {
	log.Info("Starting proposer service")
	p.shard = sharding.NewShard(big.NewInt(int64(p.shardID)), p.dbService.DB())
	p.requestsChan = make(chan *types.Transaction)
	p.txpoolSub = p.txpool.TransactionsFeed().Subscribe(p.requestsChan)
	go p.proposeCollations(p.client, p.client.SMCTransactor(), p.client.ChainReader(), p.client.SMCCaller(), p.client, p.client.Account())
	go utils.HandleServiceErrors(p.ctx.Done(), p.errChan)
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Info(fmt.Sprintf("Stopping proposer service in shard %d", p.shardID))
	defer p.cancel()
	defer close(p.requestsChan)
	defer close(p.errChan)
	p.txpoolSub.Unsubscribe()
	return nil
}

// proposeCollations listens to the transaction feed and submits collations over an interval.
func (p *Proposer) proposeCollations(manager mainchain.ContractManager, adder mainchain.RecordAdder, reader mainchain.Reader, fetcher mainchain.RecordFetcher, signer mainchain.Signer, account *accounts.Account) {
	for {
		select {
		case tx := <-p.requestsChan:

			log.Info(fmt.Sprintf("Received transaction: %x", tx.Hash()))
			blockNumber, err := reader.BlockByNumber(p.ctx, nil)
			if err != nil {
				p.errChan <- fmt.Errorf("could not fetch latest block number: %v", err)
				continue
			}
			period := new(big.Int).Div(blockNumber.Number(), big.NewInt(p.config.PeriodLength))

			// Create collation.
			collation, err := createCollation(manager, fetcher, account, signer, p.shard.ShardID(), period, []*types.Transaction{tx})
			if err != nil {
				p.errChan <- fmt.Errorf("could not create collation: %v", err)
				continue
			}

			// Saves the collation to persistent storage in the shardDB.
			if err := p.shard.SaveCollation(collation); err != nil {
				p.errChan <- fmt.Errorf("could not save collation to persistent storage: %v", err)
				continue
			}

			log.Info(fmt.Sprintf("Saved collation with header hash %v to shardChainDB", collation.Header().Hash().Hex()))

			// Check SMC if we can submit header before AddHeader to SMC.
			canAdd, err := checkHeaderAdded(fetcher, p.shard.ShardID(), period)
			if err != nil {
				p.errChan <- fmt.Errorf("could not propose collation: %v", err)
			}
			if canAdd {
				AddHeader(manager, adder, collation)
			}
		case <-p.ctx.Done():
			log.Debug("Proposer context closed, exiting goroutine")
			return
		case <-p.txpoolSub.Err():
			log.Debug("Subscriber closed")
			return
		}
	}
}
