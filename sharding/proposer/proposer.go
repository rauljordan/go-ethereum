package proposer

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/sharding"
	"github.com/ethereum/go-ethereum/sharding/mainchain"
)

// AddHeader adds the collation header to the main chain by sending
// an addHeader transaction to the sharding manager contract.
// There can only exist one header per period per shard, it is the proposer's
// responsibility to check if a header has been added.
func AddHeader(transactor mainchain.ContractTransactor, adder mainchain.RecordAdder, collation *sharding.Collation) error {
	log.Info("Adding header to SMC")

	txOps, err := transactor.CreateTXOpts(big.NewInt(0))
	if err != nil {
		return fmt.Errorf("unable to initiate add header transaction: %v", err)
	}

	// TODO: Copy is inefficient here. Let's research how to best convert hash to [32]byte.
	var chunkRoot [32]byte
	copy(chunkRoot[:], collation.Header().ChunkRoot().Bytes())

	tx, err := adder.AddHeader(txOps, collation.Header().ShardID(), collation.Header().Period(), chunkRoot, collation.Header().Sig())
	if err != nil {
		return fmt.Errorf("unable to add header to SMC: %v", err)
	}
	log.Info(fmt.Sprintf("Add header transaction hash: %v", tx.Hash().Hex()))
	return nil
}

// createCollation creates collation base struct with header
// and body. Header consists of shardID, ChunkRoot, period,
// proposer addr and signatures. Body contains serialized blob
// of a collations transactions.
func createCollation(caller mainchain.ContractCaller, fetcher mainchain.RecordFetcher, account *accounts.Account, signer mainchain.Signer, shardID *big.Int, period *big.Int, txs []*types.Transaction) (*sharding.Collation, error) {
	// shardId has to be within range
	shardCount, err := caller.GetShardCount()
	if err != nil {
		return nil, fmt.Errorf("can't get shard count from smc: %v", err)
	}
	if shardID.Cmp(big.NewInt(0)) < 0 || shardID.Cmp(big.NewInt(shardCount)) > 0 {
		return nil, fmt.Errorf("can't create collation for shard %v. Must be between 0 and %v", shardID, shardCount)
	}

	// check with SMC to see if we can add the header.
	if a, _ := checkHeaderAdded(fetcher, shardID, period); !a {
		return nil, fmt.Errorf("can't create collation, collation with same period has already been added")
	}

	// serialized tx to blob for collation body.
	blobs, err := sharding.SerializeTxToBlob(txs)
	if err != nil {
		return nil, fmt.Errorf("can't create collation, serialization to blob failed: %v", err)
	}

	// construct the header, leave chunkRoot and signature fields empty, to be filled later.
	addr := account.Address
	header := sharding.NewCollationHeader(shardID, nil, period, &addr, nil)

	// construct the body with header, blobs(serialized txs) and txs.
	collation := sharding.NewCollation(header, blobs, txs)
	collation.CalculateChunkRoot()
	sig, err := signer.Sign(collation.Header().Hash())
	if err != nil {
		return nil, fmt.Errorf("can't create collation, sign collationHeader failed: %v", err)
	}

	// add proposer signature to collation header.
	collation.Header().AddSig(sig)
	log.Info(fmt.Sprintf("Collation %v created for shardID %v period %v", collation.Header().Hash().Hex(), collation.Header().ShardID(), collation.Header().Period()))
	return collation, nil
}

// checkHeaderAdded checks if a collation header has already
// submitted to the main chain. There can only be one header per shard
// per period, proposer should check if a header's already submitted,
// checkHeaderAdded returns true if it is available, false if it is unavailable.
func checkHeaderAdded(fetcher mainchain.RecordFetcher, shardID *big.Int, period *big.Int) (bool, error) {
	// Get the period of the last header.
	lastPeriod, err := fetcher.LastSubmittedCollation(&bind.CallOpts{}, shardID)
	if err != nil {
		return false, fmt.Errorf("unable to get the period of last submitted collation: %v", err)
	}
	// True if current period is greater than last added period.
	return period.Cmp(lastPeriod) > 0, nil
}
