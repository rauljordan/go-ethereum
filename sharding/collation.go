package sharding

import (
	"fmt"
	"math"
	"math/big"
	"reflect"

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

// Transactions returns an array of tx's in the collation.
var (
	collationsizelimit = int64(math.Pow(float64(2), float64(20)))
	chunkSize          = int64(32)
	numberOfChunks     = collationsizelimit / chunkSize
)

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

// Serialize method  serializes the collation body
func (c *Collation) Serialize() ([]byte, error) {

	blob, err := utils.ConvertInterface(c.transactions, reflect.Slice)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	serializedtx, err := utils.Serialize(blob)

	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	if int64(len(serializedtx)) > collationsizelimit {
		serializedtx = serializedtx[0:collationsizelimit]

	}

	return serializedtx, nil

}

func (c *Collation) GasUsed() *big.Int {
	g := uint64(0)
	for _, tx := range c.transactions {
		if g > math.MaxUint64-(g+tx.Gas()) {
			g = math.MaxUint64
			break
		}
		g += tx.Gas()
	}
	return big.NewInt(0).SetUint64(g)
}
