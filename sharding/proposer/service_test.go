package proposer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/sharding/contracts"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/database"
	internal "github.com/ethereum/go-ethereum/sharding/internal"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
	"github.com/ethereum/go-ethereum/sharding/p2p"
	"github.com/ethereum/go-ethereum/sharding/params"
	"github.com/ethereum/go-ethereum/sharding/txpool"
)

var _ = sharding.Service(&Proposer{})

type faultyReader struct{}
type goodReader struct{}

type faultySMCCaller struct{}
type goodSMCCaller struct{}

type faultySigner struct{}
type goodSigner struct{}

func (f *faultySMCCaller) GetShardCount() (int64, error) {
	return 0, errors.New("error fetching shard count")
}

func (g *goodSMCCaller) GetShardCount() (int64, error) {
	return 100, nil
}

func (f *faultySMCCaller) SMCCaller() *contracts.SMCCaller {
	return nil
}

func (f *goodSMCCaller) SMCCaller() *contracts.SMCCaller {
	return nil
}

func (f *faultyReader) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return nil, fmt.Errorf("cannot fetch block by number")
}

func (f *faultyReader) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return nil, nil
}

func (g *goodReader) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return types.NewBlock(&types.Header{Number: big.NewInt(0)}, nil, nil, nil), nil
}

func (g *goodReader) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return nil, nil
}

func (g *goodSigner) Sign(hash common.Hash) ([]byte, error) {
	return []byte{}, nil
}

func (f *faultySigner) Sign(hash common.Hash) ([]byte, error) {
	return []byte{}, errors.New("could not sign hash")
}

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

func TestProposerCollationsFaultyReader(t *testing.T) {
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
	proposer.errChan = make(chan error)

	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{Backend: backend, SMC: smc, T: t}

	// TODO: fix this test
	go proposer.proposeCollations(client, client.SMCTransactor(), &faultyReader{}, client.SMCCaller(), client, client.Account())

	proposer.requestsChan <- &types.Transaction{}
	receivedErr := <-proposer.errChan
	expectedErr := "could not fetch latest block number"
	if !strings.Contains(receivedErr.Error(), expectedErr) {
		t.Errorf("Expected error did not match. want: %v, got: %v", expectedErr, receivedErr)
	}

	proposer.cancel()

	// The context should have been canceled.
	if proposer.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}

}

func TestProposerCollationsFaultySigner(t *testing.T) {
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

	proposer.shard = sharding.NewShard(big.NewInt(int64(shardID)), dbService.DB())

	proposer.requestsChan = make(chan *types.Transaction)
	proposer.txpoolSub = proposer.txpool.TransactionsFeed().Subscribe(proposer.requestsChan)
	proposer.errChan = make(chan error)

	backend, smc := internal.SetupMockClient(t)
	client := &internal.MockClient{Backend: backend, SMC: smc, T: t}

	// TODO: fix this test
	go proposer.proposeCollations(client, client.SMCTransactor(), &goodReader{}, client.SMCCaller(), &faultySigner{}, client.Account())

	proposer.requestsChan <- &types.Transaction{}
	receivedErr := <-proposer.errChan
	expectedErr := "could not create collation"
	if !strings.Contains(receivedErr.Error(), expectedErr) {
		t.Errorf("Expected error did not match. want: %v, got: %v", expectedErr, receivedErr)
	}

	proposer.cancel()

	// The context should have been canceled.
	if proposer.ctx.Err() == nil {
		t.Error("Context was not canceled")
	}
}

func TestProposerCollationsFaultyShard(t *testing.T) {
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

	// TODO:
}

func TestProposerCollationsFaultyCaller(t *testing.T) {
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

	// TODO:
}
