package sharding

import (
	"bytes"
	"reflect"
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

//Tests that Transactions can be serialised
func TestSerialize_Deserialize(t *testing.T) {

	tests := []struct {
		transactions []*types.Transaction
	}{
		{
			transactions: []*types.Transaction{
				makeTxWithGasLimit(0),
				makeTxWithGasLimit(5),
				makeTxWithGasLimit(20),
				makeTxWithGasLimit(100),
			},
		},
	}

	for _, tt := range tests {
		c := &Collation{}
		for _, tx := range tt.transactions {
			c.AddTransaction(tx)
		}

		tx := c.transactions

		results, err := c.Serialize()

		if err != nil {
			t.Errorf("Unable to Serialize transactions, %v", err)
		}

		deserializedTxs, err := Deserialize(results)

		if err != nil {
			t.Errorf("Unable to deserialize collation body, %v", err)
		}

		if len(tx) != len(*deserializedTxs) {
			t.Errorf("Transaction length is different before and after serialization: %v, %v", len(tx), len(*deserializedTxs))
		}

		for i := 0; i < len(tx); i++ {

			beforeSerialization := tx[i]
			afterDeserialization := (*deserializedTxs)[i]

			if beforeSerialization.Nonce() != afterDeserialization.Nonce() {

				t.Errorf("Data before serialization and after deserialization are not the same ,AccountNonce: %v, %v", beforeSerialization.Nonce(), afterDeserialization.Nonce())

			}

			if beforeSerialization.Gas() != afterDeserialization.Gas() {

				t.Errorf("Data before serialization and after deserialization are not the same ,GasLimit: %v, %v", beforeSerialization.Gas(), afterDeserialization.Gas())

			}

			if beforeSerialization.GasPrice().Cmp(afterDeserialization.GasPrice()) != 0 {

				t.Errorf("Data before serialization and after deserialization are not the same ,Price: %v, %v", beforeSerialization.GasPrice(), afterDeserialization.GasPrice())

			}

			beforeAddress := reflect.ValueOf(beforeSerialization.To())
			afterAddress := reflect.ValueOf(afterDeserialization.To())

			if reflect.DeepEqual(beforeAddress, afterAddress) {

				t.Errorf("Data before serialization and after deserialization are not the same ,Recipient: %v, %v", beforeAddress, afterAddress)

			}

			if beforeSerialization.Value().Cmp(afterDeserialization.Value()) != 0 {

				t.Errorf("Data before serialization and after deserialization are not the same ,Amount: %v, %v", beforeSerialization.Value(), afterDeserialization.Value())

			}

			beforeData := beforeSerialization.Data()
			afterData := afterDeserialization.Data()

			if !bytes.Equal(beforeData, afterData) {

				t.Errorf("Data before serialization and after deserialization are not the same ,Payload: %v, %v", beforeData, afterData)

			}

		}

	}

}

func makeTxWithGasLimit(gl uint64) *types.Transaction {
	return types.NewTransaction(0 /*nonce*/, common.HexToAddress("0x0") /*to*/, nil /*amount*/, gl, nil /*gasPrice*/, nil /*data*/)
}
