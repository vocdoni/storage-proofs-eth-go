package token

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// GetSlot returns the storage key slot for the holder.
// Position is the index slot (storage index of amount balances map).
func GetSlot(holder string, position int) ([32]byte, error) {
	var slot [32]byte
	hl, err := hex.DecodeString(trimHex(holder))
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

func trimHex(s string) string {
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	if len(s)&1 == 1 {
		s = "0" + s
	}
	return s
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}
