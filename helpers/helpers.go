package helpers

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// GetMapSlot returns the storage key slot for a holder.
// Position is the index slot (storage index of amount balances map).
func GetMapSlot(holder common.Address, position int) [32]byte {
	return crypto.Keccak256Hash(
		common.LeftPadBytes(holder[:], 32),
		common.LeftPadBytes(big.NewInt(int64(position)).Bytes(), 32),
	)
}

// ValueToBalance takes a big endian encoded value and the number of decimals
// and returns the balance as a big.Rat (considering decimals) and big.Int
// (not considering decimals).
func ValueToBalance(value []byte, decimals int) (*big.Rat, *big.Int) {
	// Parse balance value
	ibalance := new(big.Int).SetBytes(value)
	balance := BalanceToRat(ibalance, decimals)
	return balance, ibalance
}

// BalanceToRat returns the balance as a big.Rat considering the number of
// decimals.
func BalanceToRat(b *big.Int, decimals int) *big.Rat {
	return new(big.Rat).Quo(
		new(big.Rat).SetInt(b),
		new(big.Rat).SetInt(
			new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)),
	)
}

func HashFromPosition(position [32]byte) [32]byte {
	return crypto.Keccak256Hash(position[:])
}

// GetArraySlot returns the storage merkle tree key slot for a Solidity array.
// Position is the index slot (the position of the Array on the source code).
func GetArraySlot(position int) [32]byte {
	return crypto.Keccak256Hash(common.LeftPadBytes(big.NewInt(int64(position)).Bytes(), 32))
}

func ToBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}
