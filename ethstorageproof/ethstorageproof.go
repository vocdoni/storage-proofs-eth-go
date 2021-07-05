package ethstorageproof

// Initial fork from https://github.com/aergoio/aergo

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math"
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

	return VerifyProof2(proof.StateRoot, proof.Address.Bytes(), value, ProofToBytes(proof.AccountProof))
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
	return VerifyProof2(storageHash, proof.Key, value, ProofToBytes(proof.Proof))
}

func VerifyProof2(rootHash common.Hash, key []byte, value []byte, proof [][]byte) (bool, error) {
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
	// fmt.Printf("DBG VerifyProof2 (%v) -> %v %v\n", value, res, err)
	return bytes.Equal(value, res), nil
}

func RlpEncode(o RlpObject) []byte {
	return o.RlpEncode()
}

type RlpObject interface {
	RlpEncode() []byte
}

type RlpString []byte

func (s RlpString) RlpEncode() []byte {
	var rlpBytes []byte
	l := len(s)
	if l == 1 && s[0] < 0x80 {
		rlpBytes = append(rlpBytes, s[0])
	} else {
		rlpBytes = append(rlpBytes, rlpLength(l, 0x80)...)
		rlpBytes = append(rlpBytes, s...)
	}
	return rlpBytes
}

type RlpList []RlpObject

func (l RlpList) RlpEncode() []byte {
	var rlpBytes []byte
	for _, item := range l {
		rlpBytes = append(rlpBytes, item.RlpEncode()...)
	}
	length := rlpLength(len(rlpBytes), 0xc0)
	return append(length, rlpBytes...)
}

func rlpLength(dataLen int, offset byte) []byte {
	if dataLen < 56 {
		return []byte{byte(dataLen) + offset}
	} else if dataLen < math.MaxInt32 {
		var output []byte
		b := toBinary(dataLen)
		output = append(output, byte(len(b)+int(offset)+55))
		return append(output, b...)
	} else {
		return []byte{}
	}
}

func toBinary(d int) []byte {
	var b []byte
	for d > 0 {
		b = append([]byte{byte(d % 256)}, b...)
		d /= 256
	}
	return b
}

func removeHexPrefix(s string) string {
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	if len(s)&1 == 1 {
		s = "0" + s
	}
	return s
}

func ProofToBytes(proof []string) [][]byte {
	var r [][]byte
	for _, n := range proof {
		d, err := hex.DecodeString(removeHexPrefix(n))
		if err != nil {
			return [][]byte{}
		}
		r = append(r, d)
	}
	return r
}
