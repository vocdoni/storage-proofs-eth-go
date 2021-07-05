package ethstorageproof

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Bytes marshals/unmarshals as a JSON string with 0x prefix.
// The empty slice marshals as "0x".
type Bytes []byte

// MarshalText implements encoding.TextMarshaler
func (b Bytes) MarshalText() ([]byte, error) {
	result := make([]byte, len(b)*2+2)
	copy(result, `0x`)
	hex.Encode(result[2:], b)
	return result, nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *Bytes) UnmarshalText(input []byte) error {
	dec := make([]byte, len(input)/2)
	if bytes.HasPrefix(input, []byte("0x")) {
		input = input[2:]
	}
	if _, err := hex.Decode(dec, input); err != nil {
		fmt.Printf("DBG Bytes.UnmarshalText: %v\n", err)
		return err
	} else {
		*b = dec
		return nil
	}
}

type StorageProof struct {
	StateRoot    common.Hash     `json:"stateRoot"`
	Height       *big.Int        `json:"height"`
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

type StorageResult struct {
	Key   Bytes        `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// MemDB is an ethdb.KeyValueReader implementation which is not thread safe and
// assumes that all keys are common.Hash.
type MemDB struct {
	kvs map[common.Hash][]byte
}

// NewMemDB creates a new empty MemDB
func NewMemDB() *MemDB {
	return &MemDB{
		kvs: make(map[common.Hash][]byte),
	}
}

// Has returns true if the MemBD contains the key
func (m *MemDB) Has(key []byte) (bool, error) {
	var h common.Hash
	copy(h[:], key)
	_, ok := m.kvs[h]
	return ok, nil
}

// Get returns the value of the key, or nit if it's not found
func (m *MemDB) Get(key []byte) ([]byte, error) {
	var h common.Hash
	copy(h[:], key)
	value, ok := m.kvs[h]
	if ok {
		return value, nil
	} else {
		return nil, nil
	}
}

// Put sets or updates the value at key
func (m *MemDB) Put(key []byte, value []byte) error {
	var h common.Hash
	copy(h[:], key)
	m.kvs[h] = value
	return nil
}
