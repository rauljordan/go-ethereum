// Package proposer defines all relevant functionality for a Proposer actor
// within the minimal sharding protocol.
package proposer

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
)

// Proposer holds functionality required to run a collation proposer
// in a sharded system. Must satisfy the Service interface defined in
// sharding/service.go.
type Proposer struct {
	client   mainchain.Client
	shardp2p sharding.ShardP2P
	txpool   sharding.TXPool
}

// NewProposer creates a struct instance of a proposer service.
// It will have access to a mainchain client, a shardp2p network,
// and a shard transaction pool.
func NewProposer(client mainchain.Client, shardp2p sharding.ShardP2P, txpool sharding.TXPool) (*Proposer, error) {
	// Initializes a  directory persistent db.
	return &Proposer{client, shardp2p, txpool}, nil
}

// Start the main loop for proposing collations.
func (p *Proposer) Start() error {
	log.Info("Starting proposer service")
	go p.subscribeTransactions()
	return nil
}

// Stop the main loop for proposing collations.
func (p *Proposer) Stop() error {
	log.Info("Stopping proposer service")
	return nil
}

func (p *Proposer) subscribeTransactions() {
	// Subscribes to incoming transactions from the txpool via the shardp2p network.
	for {
		subchan := make(chan int)
		sub := p.txpool.TransactionsFeed().Subscribe(subchan)
		// 10 second time out for the subscription.
		timeout := time.NewTimer(10 * time.Second)
		select {
		case v := <-subchan:
			log.Info(fmt.Sprintf("Received transaction with id: %d", v))
		case <-timeout.C:
			log.Error("Receive timeout")
		}

		sub.Unsubscribe()
		select {
		case _, ok := <-sub.Err():
			if ok {
				log.Error("Channel not closed after unsubscribe")
			}
		case <-timeout.C:
			log.Error("Unsubscribe timeout")
		}
	}
}
