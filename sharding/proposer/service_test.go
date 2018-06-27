package proposer

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding/database"
	internal "github.com/ethereum/go-ethereum/sharding/internal"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/txpool"
)

func TestStop(t *testing.T) {
	h := internal.NewLogHandler(t)
	log.Root().SetHandler(h)

	shardID := 0
	server, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("Unable to setup p2p server: %v", err)
	}
	pool, err := txpool.NewTXPool(server)
	if err != nil {
		t.Fatalf("Unable to setup txpool server: %v", err)
	}
	dbService, err := database.NewShardDB("", "", true)
	if err != nil {
		t.Fatalf("Unable to setup db: %v", err)
	}

	proposer, err := NewProposer(params.DefaultConfig, &mainchain.SMCClient{}, server, pool, dbService, shardID)
	if err != nil {
		t.Fatalf("Unable to setup proposer service: %v", err)
	}

	proposer.requestsChan = make(chan *types.Transaction)
	proposer.txpoolSub = proposer.txpool.TransactionsFeed().Subscribe(proposer.requestsChan)

	if err := proposer.Stop(); err != nil {
		t.Fatalf("Unable to stop proposer service: %v", err)
	}

	h.VerifyLogMsg(fmt.Sprintf("Stopping proposer service in shard %v", shardID))

	// The context should have been canceled.
	if proposer.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
}
