package esps

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/vocdoni/storage-proofs-eth-go/helpers"
	"github.com/vocdoni/storage-proofs-eth-go/token/erc20"
)

type storageValue struct {
	address common.Hash
}

func (s *storageValue) SetStorageAddress(address []byte) {
	s.address = common.BytesToHash(address)
}

func (s *storageValue) Get(eth *erc20.ERC20Token) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10) // TODO: use a const for timeout
	defer cancel()
	return eth.Ethcli.StorageAt(ctx, common.BytesToAddress(eth.TokenAddr), s.address, nil)
}

type espmap struct {
	slot []byte
}

func (m *espmap) String() string {
	return fmt.Sprintf("slot: %x", m.slot)
}

// setSlot initializes an espmap with an indexSlot.
func (m *espmap) setSlot(indexSlot string) error {
	posHex := fmt.Sprintf("%x", indexSlot)
	if len(posHex)%2 == 1 {
		posHex = "0" + posHex
	}
	var err error
	m.slot, err = hex.DecodeString(posHex)
	if err != nil {
		return err
	}
	m.slot = common.LeftPadBytes(m.slot, 32)
	return nil
}

// getValueAddress returns the storage value address for a map key.
func (m *espmap) GetValueAddress(hexKey string) ([32]byte, error) {
	var slot [32]byte
	if len(m.slot) != 32 {
		return slot, fmt.Errorf("slot is not set for the map")
	}
	hl, err := hex.DecodeString(helpers.TrimHex(hexKey))
	if err != nil {
		return slot, err
	}
	hl = common.LeftPadBytes(hl, 32)
	hash := crypto.Keccak256(hl, m.slot)
	copy(slot[:], hash[:32])
	return slot, err
}

func VarToHex(value *espType) (string, error) {
	hexSlot := ""
	switch value.typeVar.String() {
	case "uint32", "uint64":
		hexSlot = fmt.Sprintf("%x", value.value)
	case "common.Address":
		hexSlot = value.value.(common.Address).Hex()
	case "*big.Int":
		hexSlot = fmt.Sprintf("%x", value.value.(*big.Int).Bytes())
	default:
		return "", fmt.Errorf("setMap: cannot use value type %s as map index slot", value.typeVar.String())
	}
	return hexSlot, nil
}
