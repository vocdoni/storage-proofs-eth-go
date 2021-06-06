package token

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
)

// GetMapProof returns the storage merkle proofs for the acount holder
func (w *ERC20Token) GetMapProof(ctx context.Context, holder common.Address,
	block *types.Block) (*ethstorageproof.StorageProof, error) {
	islot, _, err := w.DiscoverERC20mapSlot(holder)
	if err != nil {
		return nil, err
	}
	return w.GetMapProofWithIndexSlot(ctx, holder, block, islot)
}

// GetMapProofWithIndexSlot returns the storage merkle proofs for the acount holder.
// The index slot is the position on the EVM storage sub-trie for the contract.
// If index slot is unknown, GetProof() could be used instead to try to find it
func (w *ERC20Token) GetMapProofWithIndexSlot(ctx context.Context, holder common.Address,
	block *types.Block, islot int) (*ethstorageproof.StorageProof, error) {
	slot, err := GetMapSlot(holder.Hex(), islot)
	if err != nil {
		return nil, err
	}
	keys := []string{fmt.Sprintf("%x", slot)}
	if block == nil {
		block, err = w.GetBlock(ctx, nil)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return nil, fmt.Errorf("cannot fetch block info")
		}
	}
	return w.getProof(ctx, keys, block)
}

// DiscoverERC20mapSlot tries to find the EVM storage index slot.
// A token holder address must be provided in order to have a balance to search and compare.
// Returns ErrSlotNotFound if the slot cannot be found.
// If found, returns also the amount stored.
func (w *ERC20Token) DiscoverERC20mapSlot(holder common.Address) (int, *big.Float, error) {
	var slot [32]byte
	tokenData, err := w.GetTokenData()
	if err != nil {
		return -1, nil, fmt.Errorf("GetTokenData: %w", err)
	}
	balance, err := w.Balance(holder)
	if err != nil {
		return -1, nil, fmt.Errorf("Balance: %w", err)
	}

	addr := common.Address{}
	copy(addr[:], w.tokenAddr[:20])

	amount := big.NewFloat(0)
	index := -1
	for i := 0; i < 20; i++ {
		// Prepare storage index
		slot, err = GetMapSlot(holder.Hex(), i)
		if err != nil {
			return index, nil, fmt.Errorf("GetSlot: %w", err)
		}
		// Get Storage
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		value, err := w.Ethcli.StorageAt(ctx, addr, slot, nil)
		cancel()
		if err != nil {
			return index, nil, err
		}

		// Parse balance value
		value = common.TrimLeftZeroes(value)
		if _, ok := amount.SetString(fmt.Sprintf("0x%x", value)); !ok {
			continue
		}
		amount.Mul(amount, big.NewFloat(1/(math.Pow10(int(tokenData.Decimals)))))

		// Check if balance matches
		if amount.Cmp(balance) == 0 {
			index = i
			break
		}
	}
	if index == -1 {
		return index, nil, ErrSlotNotFound
	}
	return index, amount, nil
}
