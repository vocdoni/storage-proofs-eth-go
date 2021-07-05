package ethstorageproof

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

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
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// MemDB is an ethdb.KeyValueReader implementation which is not thread safe and
// assumes that all keys are common.Hash.
type MemDB struct {
	kvs map[common.Hash][]byte
}

func NewMemDB() *MemDB {
	return &MemDB{
		kvs: make(map[common.Hash][]byte),
	}
}

func (m *MemDB) Has(key []byte) (bool, error) {
	var h common.Hash
	copy(h[:], key)
	_, ok := m.kvs[h]
	return ok, nil
}

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

func (m *MemDB) Put(key []byte, value []byte) error {
	var h common.Hash
	copy(h[:], key)
	m.kvs[h] = value
	return nil
}
