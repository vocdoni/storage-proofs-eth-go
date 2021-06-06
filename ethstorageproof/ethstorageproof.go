package ethstorageproof

// Initial fork from https://github.com/aergoio/aergo

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
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
	for i, sp := range proof.StorageProof {
		if ok, err := VerifyEthStorageProof(&sp, proof.StorageHash); !ok {
			return false, err
		}
		fmt.Printf("Proof %d is valid\n", i)
	}
	return true, nil
}

// VerifyEthAccountProof verifies an Ethereum account proof against the StateRoot.
// It does not verify the storage proof(s).
func VerifyEthAccountProof(proof *StorageProof) (bool, error) {
	var nonce RlpString
	var balance RlpString
	var storageroot RlpString
	var codehash RlpString
	var err error

	// "0x0" means empty on RLP encoding, so if that is the case, do not decode
	if proof.Nonce.String() != "0x0" {
		nonce, err = hex.DecodeString(removeHexPrefix(proof.Nonce.String()))
		if err != nil {
			return false, err
		}
	}
	if proof.Balance.String() != "0x0" {
		balance, err = hex.DecodeString(removeHexPrefix(proof.Balance.String()))
		if err != nil {
			return false, err
		}
	}
	storageroot = proof.StorageHash.Bytes()
	codehash = proof.CodeHash.Bytes()
	values := RlpList{nonce, balance, storageroot, codehash}

	return VerifyProof(proof.Address.Bytes(), values, proof.StateRoot.Bytes(), ProofToBytes(proof.AccountProof))
}

// VerifyEthStorageProof verifies an Ethereum storage proof against the StateRoot.
// It does not verify the account proof against the Ethereum StateHash.
func VerifyEthStorageProof(proof *StorageResult, storageHash common.Hash) (bool, error) {
	var err error
	var value RlpString
	var key []byte
	if proof.Value.String() != "0x0" {
		value, err = hex.DecodeString(removeHexPrefix(proof.Value.String()))
		if err != nil {
			return false, err
		}
	} else {
		value = RlpString{0x0}
	}
	if proof.Key != "0x0" {
		key, err = hex.DecodeString(removeHexPrefix(proof.Key))
		if err != nil {
			return false, err
		}
	} else {
		return false, fmt.Errorf("proof key is nil (0x0)")
	}
	return VerifyProof(key, &value, storageHash.Bytes(), ProofToBytes(proof.Proof))
}

// VerifyProof verifies an Ethereum Merkle tree storage proof.
// This function verifies a raw proof.
func VerifyProof(key []byte, value RlpObject, expectedHash []byte, proof [][]byte) (bool, error) {
	if len(key) == 0 || len(proof) == 0 || value == nil {
		return false, fmt.Errorf("key, value or proof are empty")
	}
	key = []byte(hex.EncodeToString(keccak256(key)))
	valueRlpEncoded := RlpEncode(value)
	ks := keyStream{bytes.NewBuffer(key)}
	for i, p := range proof {
		if ((i != 0 && len(p) < 32) || !bytes.Equal(expectedHash, keccak256(p))) && !bytes.Equal(expectedHash, p) {
			return false, fmt.Errorf("expected hash does not match")
		}
		n := decodeRlpTrieNode(p)
		switch len(n) {
		case shortNode:
			if len(n[0]) == 0 {
				return false, fmt.Errorf("a short node is empty")
			}
			leaf, sharedNibbles, err := decodeHpHeader(n[0][0])
			if err != nil {
				return false, fmt.Errorf("cannot decode leaf: %w", err)
			}
			sharedNibbles = append(sharedNibbles, []byte(hex.EncodeToString(n[0][1:]))...)
			if len(sharedNibbles) == 0 {
				return false, fmt.Errorf("no nibbles in short node")
			}
			if leaf {
				if bytes.Equal(sharedNibbles, ks.key(-1)) && bytes.Equal(n[1], valueRlpEncoded) {
					return true, nil
				}
				return false, fmt.Errorf("leaf node does not match value (%x != %x)", n[1], valueRlpEncoded)
			}
			if !bytes.Equal(sharedNibbles, ks.key(len(sharedNibbles))) {
				return false, fmt.Errorf("key path does not match on short node")
			}
			expectedHash = n[1]
		case branchNode:
			if ks.Len() == 0 {
				if bytes.Equal(n[16], valueRlpEncoded) {
					return true, nil
				}
				return false, fmt.Errorf("key path ended but leaf value not found")
			}
			k := ks.index()
			if k > 0x0f {
				return false, fmt.Errorf("key path does not match on branch node")
			}
			expectedHash = n[k]
			// If last node is a branchNode, proof of not existing value
			if len(expectedHash) == 0 && bytes.Equal(valueRlpEncoded, []byte{0x00}) {
				return true, nil
			}
		default:
			return false, fmt.Errorf("unknown type of node")
		}
	}
	return false, nil
}

func decodeRlpTrieNode(data []byte) rlpNode {
	var (
		dataLen = uint64(len(data))
		node    rlpNode
	)
	if dataLen == uint64(0) {
		return nil
	}
	switch {
	case data[0] >= 0xf8:
		lenLen := int(data[0]) - 0xf7
		l, err := decodeLen(data[1:], lenLen)
		if err != nil {
			return nil
		}
		if dataLen != uint64(1)+uint64(lenLen)+l {
			return nil
		}
		node = toList(data[1+lenLen:], l)
	case data[0] >= 0xc0:
		l := uint64(data[0]) - 0xc0
		if dataLen != uint64(1+l) {
			return nil
		}
		node = toList(data[1:], l)
	}
	return node
}

func decodeLen(data []byte, lenLen int) (uint64, error) {
	if len(data) <= lenLen || lenLen > 8 {
		return 0, errDecode
	}
	switch lenLen {
	case 1:
		return uint64(data[0]), nil
	default:
		start := int(8 - lenLen)
		copy(lenBuf[:], nilBuf[:start])
		copy(lenBuf[start:], data[:lenLen])
		return binary.BigEndian.Uint64(lenBuf), nil
	}
}

func toList(data []byte, dataLen uint64) rlpNode {
	var (
		node   rlpNode
		offset = uint64(0)
	)
	for {
		e, l, err := toString(data[offset:])
		if err != nil {
			return nil
		}
		node = append(node, e)
		offset += l
		if dataLen == offset {
			break
		}
		if dataLen < offset {
			return nil
		}
	}
	nodeLen := uint64(len(node))
	if nodeLen != uint64(2) && nodeLen != uint64(17) {
		return nil
	}
	return node
}

func toString(data []byte) ([]byte, uint64, error) {
	if len(data) == 0 {
		return nil, 0, errDecode
	}
	switch {
	case data[0] <= 0x7f: // a single byte
		return data[0:1], 1, nil
	case data[0] <= 0xb7: // string <= 55
		end := 1 + data[0] - 0x80
		return data[1:end], uint64(end), nil
	case data[0] <= 0xbf: // string > 55
		lenLen := data[0] - 0xb7
		l, err := decodeLen(data[1:], int(lenLen))
		if err != nil {
			return nil, 0, err
		}
		start := 1 + lenLen
		end := uint64(start) + l
		return data[start:end], end, nil
	default:
		return nil, 0, errDecode
	}
}

func keccak256(data ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	for _, d := range data {
		h.Write(d)
	}
	return h.Sum(nil)
}

func keccak256Hex(data ...[]byte) string {
	return hex.EncodeToString(keccak256(data...))
}

func decodeHpHeader(b byte) (bool, []byte, error) {
	switch b >> 4 {
	case 0:
		return false, []byte{}, nil
	case 1:
		return false, []byte{hexChar[b&0x0f]}, nil
	case 2:
		return true, []byte{}, nil
	case 3:
		return true, []byte{hexChar[b&0x0f]}, nil
	default:
		return false, []byte{}, errDecode
	}
}

func hexToIndex(c byte) (byte, error) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', nil
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, nil
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, nil
	}
	return 0, errDecode
}

func (ks keyStream) index() byte {
	b, err := ks.ReadByte()
	if err != nil {
		return 0x10
	}
	i, err := hexToIndex(b)
	if err != nil {
		return 0x10
	}
	return i
}

func (ks keyStream) key(l int) []byte {
	if l == -1 {
		return ks.Buffer.Bytes()
	}
	return ks.Buffer.Next(l)
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
