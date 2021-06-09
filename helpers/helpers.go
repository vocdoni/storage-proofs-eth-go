package helpers

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// GetMapSlot returns the storage key slot for a holder.
// Position is the index slot (storage index of amount balances map).
func GetMapSlot(holder string, position int) ([32]byte, error) {
	var slot [32]byte
	hl, err := hex.DecodeString(TrimHex(holder))
	if err != nil {
		return slot, err
	}
	hl = common.LeftPadBytes(hl, 32)
	posHex := fmt.Sprintf("%x", position)
	if len(posHex)%2 == 1 {
		posHex = "0" + posHex
	}
	p, err := hex.DecodeString(posHex)
	if err != nil {
		return slot, err
	}
	p = common.LeftPadBytes(p, 32)

	hash := crypto.Keccak256(hl, p)
	copy(slot[:], hash[:32])
	return slot, err
}

func HashFromPosition(position string) ([32]byte, error) {
	var slot [32]byte
	hl, err := hex.DecodeString(TrimHex(position))
	if err != nil {
		return slot, err
	}
	hl = common.LeftPadBytes(hl, 32)
	hash := crypto.Keccak256(hl)
	copy(slot[:], hash[:32])
	return slot, err
}

// GetArraySlot returns the storage merkle tree key slot for a Solidity array.
// Position is the index slot (the position of the Array on the source code).
func GetArraySlot(position int) ([32]byte, error) {
	var slot [32]byte
	posHex := fmt.Sprintf("%x", position)
	if len(posHex)%2 == 1 {
		posHex = "0" + posHex
	}
	p, err := hex.DecodeString(posHex)
	if err != nil {
		return slot, err
	}
	p = common.LeftPadBytes(p, 32)

	hash := crypto.Keccak256(p)
	copy(slot[:], hash[:32])
	return slot, err
}

func TrimHex(s string) string {
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	if len(s)&1 == 1 {
		s = "0" + s
	}
	return s
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
