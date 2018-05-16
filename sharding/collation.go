package sharding

import (
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/sharding/utils"
)

// Collation base struct.
type Collation struct {
	header       *CollationHeader
	transactions []*types.Transaction
}

// CollationHeader base struct.
type CollationHeader struct {
	shardID           *big.Int        //the shard ID of the shard.
	chunkRoot         *common.Hash    //the root of the chunk tree which identifies collation body.
	period            *big.Int        //the period number in which collation to be included.
	proposerAddress   *common.Address //address of the collation proposer.
	proposerSignature []byte          //the proposer's signature for calculating collation hash.
}

// Header returns the collation's header.
func (c *Collation) Header() *CollationHeader { return c.header }

var (
	collationSizelimit = int64(math.Pow(float64(2), float64(20)))
	chunkSize          = int64(32)
	numberOfChunks     = collationSizelimit / chunkSize
)

// Transactions returns an array of tx's in the collation.
func (c *Collation) Transactions() []*types.Transaction { return c.transactions }

// ShardID is the identifier for a shard.
func (c *Collation) ShardID() *big.Int { return c.header.shardID }

// Period the collation corresponds to.
func (c *Collation) Period() *big.Int { return c.header.period }

// ProposerAddress is the coinbase addr of the creator for the collation.
func (c *Collation) ProposerAddress() *common.Address { return c.header.proposerAddress }

// SetHeader updates the collation's header.
func (c *Collation) SetHeader(h *CollationHeader) { c.header = h }

// AddTransaction adds to the collation's body of tx blobs.
func (c *Collation) AddTransaction(tx *types.Transaction) {
	// TODO: Include blob serialization instead.
	c.transactions = append(c.transactions, tx)
}

// CreateRawBlobs creates raw blobs from transactions.
func (c *Collation) CreateRawBlobs() ([]*utils.RawBlob, error) {

	// It does not skip evm execution by default
	blobs := make([]*utils.RawBlob, len(c.transactions))
	for i := 0; i < len(c.transactions); i++ {

		err := error(nil)
		blobs[i], err = utils.NewRawBlob(c.transactions[i], false)

		if err != nil {
			return nil, fmt.Errorf("Creation of raw blobs from transactions failed %v", err)
		}

	}

	return blobs, nil

}

// ConvertBackToTx converts raw blobs back to their original transactions.
func ConvertBackToTx(rawBlobs []utils.RawBlob) ([]*types.Transaction, error) {

	blobs := make([]*types.Transaction, len(rawBlobs))

	for i := 0; i < len(rawBlobs); i++ {

		blobs[i] = types.NewTransaction(0, common.HexToAddress("0x"), nil, 0, nil, nil)

		err := utils.ConvertFromRawBlob(&rawBlobs[i], blobs[i])
		if err != nil {
			return nil, fmt.Errorf("Creation of transactions from raw blobs failed %v", err)
		}
	}
	return blobs, nil

}

// Serialize method  serializes the collation body to a byte array.
func (c *Collation) Serialize() ([]byte, error) {

	blobs, err := c.CreateRawBlobs()

	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	serializedTx, err := utils.Serialize(blobs)

	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	if int64(len(serializedTx)) > collationSizelimit {

		return nil, fmt.Errorf("The serialized body exceeded the collation size limit: %v", serializedTx)

	}

	return serializedTx, nil

}

// Deserialize takes a byte array and converts its back to its original transactions.
func Deserialize(serialisedBlob []byte) (*[]*types.Transaction, error) {

	deserializedBlobs, err := utils.Deserialize(serialisedBlob)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	txs, err := ConvertBackToTx(deserializedBlobs)

	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	return &txs, nil
}
