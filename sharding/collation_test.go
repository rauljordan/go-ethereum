package sharding

import (
	//"github.com/ethereum/go-ethereum/rlp"
	//"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestCollation_AddTransactions(t *testing.T) {
	tests := []struct {
		transactions []*types.Transaction
	}{
		{
			transactions: []*types.Transaction{
				makeTxWithGasLimit(0),
				makeTxWithGasLimit(1),
				makeTxWithGasLimit(2),
				makeTxWithGasLimit(3),
			},
		}, {
			transactions: []*types.Transaction{},
		},
	}

	for _, tt := range tests {
		c := &Collation{}
		for _, tx := range tt.transactions {
			c.AddTransaction(tx)
		}
		results := c.Transactions()
		if len(results) != len(tt.transactions) {
			t.Fatalf("Wrong number of transactions. want=%d. got=%d", len(tt.transactions), len(results))
		}
		for i, tx := range tt.transactions {
			if results[i] != tx {
				t.Fatalf("Mismatched transactions. wanted=%+v. got=%+v", tt.transactions, results)
			}
		}
	}
}

//TODO: Add test for converting *types.Transaction into raw blobs

//Tests thta Transactions can be serialised
func TestSerialize(t *testing.T) {
	tests := []struct {
		transactions []*types.Transaction
	}{
		{
			transactions: []*types.Transaction{
				makeTxWithGasLimit(0),
				makeTxWithGasLimit(1),
				makeTxWithGasLimit(2),
				makeTxWithGasLimit(3),
			},
		}, {
			transactions: []*types.Transaction{},
		},
	}

	for _, tt := range tests {
		c := &Collation{}
		for _, tx := range tt.transactions {
			c.AddTransaction(tx)
		}

		/*var tests *types.Transaction
		yadd := reflect.ValueOf(*c.transactions[3])
		d := yadd.FieldByName("data").FieldByName("Hash")

		test, err := rlp.EncodeToBytes(c.transactions[3])
		if err != nil {
			t.Fatalf("%v\n %v\n %v", err, test, *(c.transactions[0]))
		}
		erx := rlp.DecodeBytes(test, &tests)

		dd := reflect.ValueOf(*tests)
		cv := dd.FieldByName("data").FieldByName("Hash")

		if cv != d {
			t.Fatalf("%v\n %v\n %v", erx, cv, d)
		} */

		results, err := c.Serialize()
		if err == nil {
			t.Fatalf("%v\n %v\n %v", err, results, c.transactions)
		}

	}

}

func makeTxWithGasLimit(gl uint64) *types.Transaction {
	return types.NewTransaction(0 /*nonce*/, common.HexToAddress("0x0") /*to*/, nil /*amount*/, gl, nil /*gasPrice*/, nil /*data*/)
}
