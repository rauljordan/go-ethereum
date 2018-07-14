package mainchain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	log "github.com/sirupsen/logrus"
)

// Reader defines a struct that can fetch latest header events from a web3 endpoint.
type Reader interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error)
}

// Web3Service fetches important information about the canonical
// Ethereum PoW chain via a web3 endpoint using an ethclient. The Random
// Beacon Chain requires synchronization with the mainchain's current
// blockhash, block number, and access to logs within the
// Validator Registration Contract on the mainchain to kick off the beacon
// chain's validator registration process.
type Web3Service struct {
	ctx         context.Context
	cancel      context.CancelFunc
	headerChan  chan *gethTypes.Header
	endpoint    string
	blockNumber *big.Int    // the latest mainchain blocknumber.
	blockHash   common.Hash // the latest mainchain blockhash.
}

// NewWeb3Service sets up a new instance with an ethclient when
// given a web3 endpoint as a string.
func NewWeb3Service(endpoint string) (*Web3Service, error) {
	if !strings.HasPrefix(endpoint, "ws") && !strings.HasPrefix(endpoint, "ipc") {
		return nil, fmt.Errorf("web3service requires either an IPC or WebSocket endpoint, provided %s", endpoint)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Web3Service{
		ctx:         ctx,
		cancel:      cancel,
		headerChan:  make(chan *gethTypes.Header),
		endpoint:    endpoint,
		blockNumber: nil,
		blockHash:   common.BytesToHash([]byte{}),
	}, nil
}

// Start a web3 service's main event loop.
func (w *Web3Service) Start() {
	log.Infof("Starting web3 mainchain service at %s", w.endpoint)
	rpcClient, err := rpc.Dial(w.endpoint)
	if err != nil {
		log.Errorf("Cannot connect to RPC client: %v", err)
		return
	}
	client := ethclient.NewClient(rpcClient)
	go w.latestMainchainInfo(client, w.ctx.Done())
}

// Stop the web3 service's main event loop and associated goroutines.
func (w *Web3Service) Stop() error {
	defer w.cancel()
	defer close(w.headerChan)
	log.Info("Stopping web3 mainchain service")
	return nil
}

func (w *Web3Service) latestMainchainInfo(reader Reader, done <-chan struct{}) {
	if _, err := reader.SubscribeNewHead(w.ctx, w.headerChan); err != nil {
		log.Errorf("Unable to subscribe to incoming headers: %v", err)
		return
	}
	for {
		select {
		case <-done:
			return
		case header := <-w.headerChan:
			w.blockNumber = header.Number
			w.blockHash = header.Hash()
			log.Debugf("Latest mainchain blocknumber: %v", w.blockNumber)
			log.Debugf("Latest mainchain blockhash: %v", w.blockHash.Hex())
		}
	}
}

// LatestBlockNumber is a getter for blockNumber to make it read-only.
func (w *Web3Service) LatestBlockNumber() *big.Int {
	return w.blockNumber
}

// LatestBlockHash is a getter for blockHash to make it read-only.
func (w *Web3Service) LatestBlockHash() common.Hash {
	return w.blockHash
}
