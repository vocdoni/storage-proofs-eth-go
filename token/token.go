package token

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	"github.com/vocdoni/storage-proofs-eth-go/token/mapbased"
	"github.com/vocdoni/storage-proofs-eth-go/token/minime"
)

const (
	TokenTypeMapbased = iota
	TokenTypeMinime
)

type Token interface {
	DiscoverSlot(ctx context.Context, holder common.Address) (int, *big.Rat, error)
	GetProof(ctx context.Context, holder common.Address, block *big.Int,
		indexSlot int) (*ethstorageproof.StorageProof, error)
	VerifyProof(holder common.Address, storageRoot common.Hash,
		proofs []ethstorageproof.StorageResult, indexSlot int, targetBalance,
		targetBlock *big.Int) error
}

func New(ctx context.Context, rpcCli *rpc.Client, tokenType int,
	address common.Address) (Token, error) {
	switch tokenType {
	case TokenTypeMapbased:
		return mapbased.New(ctx, rpcCli, address)
	case TokenTypeMinime:
		return minime.New(ctx, rpcCli, address)
	default:
		return nil, fmt.Errorf("tokentype %d unknown", tokenType)
	}
}
