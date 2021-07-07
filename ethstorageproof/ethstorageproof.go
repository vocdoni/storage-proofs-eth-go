package ethstorageproof

// Initial fork from https://github.com/aergoio/aergo

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// VerifyEIP1186 verifies the whole Ethereum proof obtained with eth_getProof
// method against a StateRoot.  It verifies Account proof against StateRoot and
// all Storage proofs against StorageHash.
func VerifyEIP1186(proof *StorageProof) (bool, error) {
	for _, sp := range proof.StorageProof {
		sp := sp
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
		proof.Nonce, proof.Balance.ToInt(), proof.StorageHash, proof.CodeHash,
	})
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

	if len(proof.Value) != 0 {
		value, err = rlp.EncodeToBytes(proof.Value)
		if err != nil {
			return false, err
		}
	}
	return VerifyProof(storageHash, proof.Key, value, proof.Proof)
}

// VerifyProof verifies that the path generated from key, following the nodes
// in proof leads to a leaf with value, where the hashes are correct up to the
// rootHash.
// WARNING: When the value is not found, `eth_getProof` will return "0x0" at
// the StorageProof `value` field.  In order to verify the proof of non
// existence, you must set `value` to nil, *not* the RLP encoding of 0 or null
// (which would be 0x80).
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
	return bytes.Equal(value, res), nil
}
