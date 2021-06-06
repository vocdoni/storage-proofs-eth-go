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

func (w *ERC20Token) GetMinimeAtPosition(holder common.Address, mapIndexSlot,
	position int) (*big.Float, *big.Int, *common.Hash, error) {
	token, err := w.GetTokenData()
	if err != nil {
		return nil, nil, nil, err
	}
	contractAddr := common.Address{}
	copy(contractAddr[:], w.tokenAddr[:20])

	mapSlot, err := GetMapSlot(holder.Hex(), mapIndexSlot)
	if err != nil {
		return nil, nil, nil, err
	}
	vf, err := HashFromPosition(fmt.Sprintf("%x", mapSlot))
	if err != nil {
		return nil, nil, nil, err
	}

	offset := new(big.Int).SetInt64(int64(position - 1))
	v := new(big.Int).SetBytes(vf[:])
	v.Add(v, offset)

	arraySlot := common.BytesToHash(v.Bytes())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	value, err := w.Ethcli.StorageAt(ctx, contractAddr, arraySlot, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	block := new(big.Int).SetBytes(common.TrimLeftZeroes(value[16:]))
	amount := new(big.Float)
	if _, ok := amount.SetString(fmt.Sprintf("0x%x", value[:16])); !ok {
		return nil, nil, nil, fmt.Errorf("amount cannot be parsed")
	}
	amount.Mul(amount, big.NewFloat(1/(math.Pow10(int(token.Decimals)))))

	return amount, block, &arraySlot, nil
}

func (w *ERC20Token) GetMinimeProofForBlock(holder common.Address,
	block *types.Block, islot int) (*ethstorageproof.StorageProof, error) {
	if block == nil {
		return nil, fmt.Errorf("block is nil")
	}
	size, err := w.GetMinimeArraySize(holder, islot)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch minime array size: %w", err)
	}

	// Check first the last block
	_, mblock, slot, err := w.GetMinimeAtPosition(holder, islot, size)
	if err != nil {
		return nil, fmt.Errorf("cannot get minime: %w", err)
	}
	if block.NumberU64() > mblock.Uint64() {
		_, _, slot2, err := w.GetMinimeAtPosition(holder, islot, size+1)
		if err != nil {
			return nil, err
		}
		keys := []string{fmt.Sprintf("%x", slot), fmt.Sprintf("%x", slot2)}
		return w.getProof(context.Background(), keys, block)
	}

	return nil, fmt.Errorf("not implemented")
}

func (w *ERC20Token) GetMinimeArraySize(holder common.Address, islot int) (int, error) {
	// In this slot we should find the array size
	mapSlot, err := GetMapSlot(holder.Hex(), islot)
	if err != nil {
		return 0, err
	}

	addr := common.Address{}
	copy(addr[:], w.tokenAddr[:20])

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	value, err := w.Ethcli.StorageAt(ctx, addr, mapSlot, nil)
	cancel()
	if err != nil {
		return 0, err
	}
	return int(new(big.Int).SetBytes(common.TrimLeftZeroes(value)).Uint64()), nil
}

func (w *ERC20Token) DiscoverMinimeSlot(holder common.Address) (int, *big.Float, error) {
	balance, err := w.Balance(holder)
	if err != nil {
		return -1, nil, fmt.Errorf("Balance: %w", err)
	}

	addr := common.Address{}
	copy(addr[:], w.tokenAddr[:20])
	amount := big.NewFloat(0)
	block := new(big.Int)
	index := -1

	for i := 0; i < 20; i++ {
		checkPointsSize, err := w.GetMinimeArraySize(holder, i)
		if err != nil {
			return 0, nil, err
		}
		if checkPointsSize <= 0 {
			continue
		}

		fmt.Printf("found a possible checkPoint array with size %d\n", checkPointsSize)

		if amount, block, _, err = w.GetMinimeAtPosition(
			holder,
			i,
			checkPointsSize,
		); err != nil {
			continue
		}
		if block.Uint64() == 0 {
			continue
		}

		// Check if balance matches
		a, _ := amount.Uint64()
		if b, _ := balance.Uint64(); b == a {
			index = i
			break
		}
	}
	if index == -1 {
		return index, nil, ErrSlotNotFound
	}
	return index, amount, nil
}
