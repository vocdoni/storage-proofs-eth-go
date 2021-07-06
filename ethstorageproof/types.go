package ethstorageproof

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// BytesHex marshals/unmarshals as a JSON string in hex with 0x prefix.  The empty
// slice marshals as "0x".
type BytesHex []byte

// MarshalText implements encoding.TextMarshaler
func (b BytesHex) MarshalText() ([]byte, error) {
	result := make([]byte, len(b)*2+2)
	copy(result, `0x`)
	hex.Encode(result[2:], b)
	return result, nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *BytesHex) UnmarshalText(input []byte) error {
	if bytes.HasPrefix(input, []byte("0x")) {
		input = input[2:]
	}
	dec := make([]byte, len(input)/2)
	if _, err := hex.Decode(dec, input); err != nil {
		return err
	} else {
		*b = dec
		return nil
	}
}

// SliceBytesHex marshals/unmarshals as a JSON vector of strings with in hex with
// 0x prefix.
type SliceBytesHex [][]byte

// MarshalText implements encoding.TextMarshaler
func (s SliceBytesHex) MarshalJSON() ([]byte, error) {
	bs := make([]BytesHex, len(s))
	for i, b := range s {
		bs[i] = b
	}
	return json.Marshal(bs)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *SliceBytesHex) UnmarshalJSON(data []byte) error {
	var bs []BytesHex
	if err := json.Unmarshal(data, &bs); err != nil {
		return err
	}
	*s = make([][]byte, len(bs))
	for i, b := range bs {
		(*s)[i] = b
	}
	return nil
}

// StorageProof allows unmarshaling the object returned by `eth_getProof`:
// https://eips.ethereum.org/EIPS/eip-1186
type StorageProof struct {
	StateRoot    common.Hash     `json:"stateRoot"`
	Height       *big.Int        `json:"height"`
	Address      common.Address  `json:"address"`
	AccountProof SliceBytesHex   `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

// StorageResult is an object from StorageProof that contains a proof of
// storage.
type StorageResult struct {
	Key   BytesHex      `json:"key"`
	Value *hexutil.Big  `json:"value"`
	Proof SliceBytesHex `json:"proof"`
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
