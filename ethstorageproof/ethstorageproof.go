package ethstorageproof

// Initial fork from https://github.com/aergoio/aergo

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	shortNode  = 2
	branchNode = 17
	hexChar    = "0123456789abcdef"
)

type (
	rlpNode   [][]byte
	keyStream struct {
		*bytes.Buffer
	}
)

var (
	errDecode = errors.New("storage proof decode error")
	lenBuf    = make([]byte, 8)
	nilBuf    = make([]byte, 8)
)

// VerifyEIP1186 verifies the whole Ethereum proof obtained with eth_getProof method against a StateRoot.
// It verifies Account proof against StateRoot and all Storage proofs against StorageHash.
func VerifyEIP1186(proof *StorageProof) (bool, error) {
	for _, sp := range proof.StorageProof {
		if ok, err := VerifyEthStorageProof(&sp, proof.StorageHash); !ok {
			return false, err
		}
	}
	return true, nil
}

// VerifyEthAccountProof verifies an Ethereum account proof against the StateRoot.
// It does not verify the storage proof(s).
func VerifyEthAccountProof(proof *StorageProof) (bool, error) {
	value, err := rlp.EncodeToBytes([]interface{}{
		proof.Nonce, proof.Balance, proof.StorageHash, proof.CodeHash})
	if err != nil {
		return false, err
	}

	return VerifyProof(proof.StateRoot, proof.Address.Bytes(), value, proof.AccountProof)
}

// VerifyEthStorageProof verifies an Ethereum storage proof against the StateRoot.
// It does not verify the account proof against the Ethereum StateHash.
func VerifyEthStorageProof(proof *StorageResult, storageHash common.Hash) (bool, error) {
	var err error
	var value []byte

	v := big.Int(*proof.Value)
	if v.BitLen() != 0 {
		value, err = rlp.EncodeToBytes(&v)
		if err != nil {
			return false, err
		}
	}
	return VerifyProof(storageHash, proof.Key, value, proof.Proof)
}

func VerifyProof(rootHash common.Hash, key []byte, value []byte, proof [][]byte) (bool, error) {
	proofDB := NewMemDB()
	for _, node := range proof {
		key := crypto.Keccak256(node)
		proofDB.Put(key, node)
	}
	path := crypto.Keccak256(key)

	res, err := trie.VerifyProof(rootHash, path, proofDB)
	if err != nil {
		return false, err
	}
	// fmt.Printf("DBG VerifyProof (%v) -> %v %v\n", value, res, err)
	return bytes.Equal(value, res), nil
}
