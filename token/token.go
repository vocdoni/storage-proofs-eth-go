package token

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	"github.com/vocdoni/storage-proofs-eth-go/token/mapbased"
	"github.com/vocdoni/storage-proofs-eth-go/token/minime"
)

const (
	TokenTypeMapbased = iota
	TokenTypeMinime
)

type Token interface {
	Init(tokenAddress, web3endpoint string) error
	DiscoverSlot(holder common.Address) (int, *big.Float, error)
	GetProof(holder common.Address, block *big.Int, indexSlot int) (*ethstorageproof.StorageProof, error)
	GetBlock(block *big.Int) (*types.Block, error)
	VerifyProof(holder common.Address, storageRoot common.Hash, proofs []ethstorageproof.StorageResult,
		indexSlot int, targetBalance, targetBlock *big.Int) error
}

func NewToken(tokenType int, address, web3endpoint string) (Token, error) {
	var t Token
	switch tokenType {
	case TokenTypeMapbased:
		t = new(mapbased.Mapbased)
		if err := t.Init(address, web3endpoint); err != nil {
			return nil, err
		}
	case TokenTypeMinime:
		t = new(minime.Minime)
		if err := t.Init(address, web3endpoint); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("tokentype %d unknown", tokenType)
	}
	return t, nil
}
