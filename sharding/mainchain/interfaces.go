package mainchain

import (
	"context"
	"math/big"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/sharding/contracts"
)

// Signer defines an interface that can read from the Ethereum mainchain as well call
// read-only methods and functions from the Sharding Manager Contract.
type Signer interface {
	Sign(hash common.Hash) ([]byte, error)
}

// ContractManager specifies an interface that defines both read/write
// operations on a contract in the Ethereum mainchain.
type ContractManager interface {
	ContractCaller
	ContractTransactor
}

// ContractCaller defines an interface that can read from a contract on the
// Ethereum mainchain as well as call its read-only methods and functions.
type ContractCaller interface {
	SMCCaller() *contracts.SMCCaller
	GetShardCount() (int64, error)
}

// ContractTransactor defines an interface that can transact with a contract on the
// Ethereum mainchain as well as call its methods and functions.
type ContractTransactor interface {
	SMCTransactor() *contracts.SMCTransactor
	CreateTXOpts(value *big.Int) (*bind.TransactOpts, error)
}

// EthClient defines the methods that will be used to perform rpc calls
// to the main geth node, and be responsible for other user-specific data
type EthClient interface {
	Account() *accounts.Account
	WaitForTransaction(ctx context.Context, hash common.Hash, durationInSeconds time.Duration) error
	TransactionReceipt(hash common.Hash) (*types.Receipt, error)
	DepositFlag() bool
}

// Reader defines an interface for a struct that can read mainchain information
// such as blocks, transactions, receipts, and more. Useful for testing.
type Reader interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

// RecordFetcher serves as an interface for a struct that can fetch collation information
// from a sharding manager contract on the Ethereum mainchain.
type RecordFetcher interface {
	CollationRecords(opts *bind.CallOpts, shardID *big.Int, period *big.Int) (struct {
		ChunkRoot [32]byte
		Proposer  common.Address
		IsElected bool
		Signature []byte
	}, error)
	LastSubmittedCollation(opts *bind.CallOpts, shardID *big.Int) (*big.Int, error)
}

// RecordAdder serves as an interface for a struct that can add collation information
// to a sharding manager contract on the Ethereum mainchain.
type RecordAdder interface {
	AddHeader(opts *bind.TransactOpts, shardID *big.Int, period *big.Int, chunkRoot [32]byte, signature []byte) (*types.Transaction, error)
}
