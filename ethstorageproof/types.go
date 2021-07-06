package ethstorageproof

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// QuantityBytes marshals/unmarshals as a JSON string in hex with 0x prefix encoded
// as a QUANTITY.  The empty slice marshals as "0x0".
type QuantityBytes []byte

// MarshalText implements encoding.TextMarshaler
func (q QuantityBytes) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("0x%v",
		new(big.Int).SetBytes(q).Text(16))), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (q *QuantityBytes) UnmarshalText(input []byte) error {
	input = bytes.TrimPrefix(input, []byte("0x"))
	v, ok := new(big.Int).SetString(string(input), 16)
	if !ok {
		return fmt.Errorf("invalid hex input")
	}
	*q = v.Bytes()
	return nil
}

// SliceData marshals/unmarshals as a JSON vector of strings with in hex with
// 0x prefix.
type SliceData [][]byte

// MarshalText implements encoding.TextMarshaler
func (s SliceData) MarshalJSON() ([]byte, error) {
	bs := make([]hexutil.Bytes, len(s))
	for i, b := range s {
		bs[i] = b
	}
	return json.Marshal(bs)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (s *SliceData) UnmarshalJSON(data []byte) error {
	var bs []hexutil.Bytes
	if err := json.Unmarshal(data, &bs); err != nil {
		return err
	}
	*s = make([][]byte, len(bs))
	for i, b := range bs {
		(*s)[i] = b
	}
	return nil
}

// StorageProof allows unmarshaling the object returned by `eth_getProof`.
// From https://eips.ethereum.org/EIPS/eip-1186:
//
//  Parameters
//
//     DATA, 20 Bytes - address of the account.
//     ARRAY, 32 Bytes - array of storage-keys which should be proofed and
//       included. See eth_getStorageAt
//     QUANTITY|TAG - integer block number, or the string "latest" or
//       "earliest", see the default block parameter
//
// Returns
//
// Object - A account object:
//
//     balance: QUANTITY - the balance of the account. See eth_getBalance
//     codeHash: DATA, 32 Bytes - hash of the code of the account. For a simple
//       Account without code it will return
//       "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
//     nonce: QUANTITY, - nonce of the account. See eth_getTransactionCount
//     storageHash: DATA, 32 Bytes - SHA3 of the StorageRoot. All storage will
//       deliver a MerkleProof starting with this rootHash.
//     accountProof: ARRAY - Array of rlp-serialized MerkleTree-Nodes, starting
//       with the stateRoot-Node, following the path of the SHA3 (address) as
//       key.
//
//     storageProof: ARRAY - Array of storage-entries as requested. Each entry is a object with these properties:
//         key: QUANTITY - the requested storage key
//         value: QUANTITY - the storage value
//         proof: ARRAY - Array of rlp-serialized MerkleTree-Nodes, starting
//           with the storageHash-Node, following the path of the SHA3 (key) as
//           path.
//
// NOTE: QUANTITY is supposed to follow this spec:
// https://infura.io/docs/ethereum#section/Value-encoding/Quantity but
// go-ethereum sometimes gives the string without the `0x` prefix
type StorageProof struct {
	Height       *big.Int        `json:"height"`
	Address      common.Address  `json:"address"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StateRoot    common.Hash     `json:"stateRoot"`
	StorageHash  common.Hash     `json:"storageHash"`
	AccountProof SliceData       `json:"accountProof"`
	StorageProof []StorageResult `json:"storageProof"`
}

// StorageResult is an object from StorageProof that contains a proof of
// storage.
type StorageResult struct {
	Key   QuantityBytes `json:"key"`
	Value QuantityBytes `json:"value"`
	Proof SliceData     `json:"proof"`
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
	h := common.BytesToHash(key)
	_, ok := m.kvs[h]
	return ok, nil
}

// Get returns the value of the key, or nil if it's not found
func (m *MemDB) Get(key []byte) ([]byte, error) {
	h := common.BytesToHash(key)
	value, ok := m.kvs[h]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	return value, nil
}

// Put sets or updates the value at key
func (m *MemDB) Put(key []byte, value []byte) error {
	h := common.BytesToHash(key)
	m.kvs[h] = value
	return nil
}
