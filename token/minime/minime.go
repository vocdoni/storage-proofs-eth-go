package minime

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	"github.com/vocdoni/storage-proofs-eth-go/helpers"
	"github.com/vocdoni/storage-proofs-eth-go/token/erc20"
)

// ErrSlotNotFound represents the storage slot not found error
var ErrSlotNotFound = errors.New("storage slot not found")

const maxIterationsForDiscover = 20

// Minime token stores the whole list of balances an address has had.
// To this end we need to generate two proofs, one for proving the balance
// on a specific block and the following proving the next balance stored
// is either nil (0x0) or a bigger block number.
type Minime struct {
	erc20 *erc20.ERC20Token
}

func (m *Minime) Init(tokenAddress, web3endpoint string) error {
	m.erc20 = &erc20.ERC20Token{}
	return m.erc20.Init(context.Background(), web3endpoint, tokenAddress)
}

func (m *Minime) GetBlock(block *big.Int) (*types.Block, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return m.erc20.GetBlock(ctx, block)
}

func (m *Minime) DiscoverSlot(holder common.Address) (int, *big.Float, error) {
	balance, err := m.erc20.Balance(holder)
	if err != nil {
		return -1, nil, err
	}

	addr := common.Address{}
	copy(addr[:], m.erc20.TokenAddr[:20])
	amount := big.NewFloat(0)
	block := new(big.Int)
	index := -1

	for i := 0; i < maxIterationsForDiscover; i++ {
		checkPointsSize, err := m.getMinimeArraySize(holder, i)
		if err != nil {
			return 0, nil, err
		}
		if checkPointsSize <= 0 {
			continue
		}

		if amount, block, _, err = m.getMinimeAtPosition(
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

func (m *Minime) GetProof(holder common.Address, block *big.Int,
	islot int) (*ethstorageproof.StorageProof, error) {
	blockData, err := m.GetBlock(block)
	if err != nil {
		return nil, err
	}
	size, err := m.getMinimeArraySize(holder, islot)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch minime array size: %w", err)
	}

	// Check first the last block
	_, mblock, slot, err := m.getMinimeAtPosition(holder, islot, size)
	if err != nil {
		return nil, fmt.Errorf("cannot get minime: %w", err)
	}
	if blockData.NumberU64() > mblock.Uint64() {
		_, _, slot2, err := m.getMinimeAtPosition(holder, islot, size+1)
		if err != nil {
			return nil, err
		}
		keys := []string{fmt.Sprintf("%x", slot), fmt.Sprintf("%x", slot2)}
		return m.erc20.GetProof(context.Background(), keys, blockData)
	}

	return nil, fmt.Errorf("not implemented")
}

func (m *Minime) getMinimeAtPosition(holder common.Address, mapIndexSlot,
	position int) (*big.Float, *big.Int, *common.Hash, error) {
	token, err := m.erc20.GetTokenData()
	if err != nil {
		return nil, nil, nil, err
	}
	contractAddr := common.Address{}
	copy(contractAddr[:], m.erc20.TokenAddr[:20])

	mapSlot, err := helpers.GetMapSlot(holder.Hex(), mapIndexSlot)
	if err != nil {
		return nil, nil, nil, err
	}
	vf, err := helpers.HashFromPosition(fmt.Sprintf("%x", mapSlot))
	if err != nil {
		return nil, nil, nil, err
	}

	offset := new(big.Int).SetInt64(int64(position - 1))
	v := new(big.Int).SetBytes(vf[:])
	v.Add(v, offset)

	arraySlot := common.BytesToHash(v.Bytes())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	value, err := m.erc20.Ethcli.StorageAt(ctx, contractAddr, arraySlot, nil)
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

func (m *Minime) getMinimeArraySize(holder common.Address, islot int) (int, error) {
	// In this slot we should find the array size
	mapSlot, err := helpers.GetMapSlot(holder.Hex(), islot)
	if err != nil {
		return 0, err
	}

	addr := common.Address{}
	copy(addr[:], m.erc20.TokenAddr[:20])

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	value, err := m.erc20.Ethcli.StorageAt(ctx, addr, mapSlot, nil)
	cancel()
	if err != nil {
		return 0, err
	}
	return int(new(big.Int).SetBytes(common.TrimLeftZeroes(value)).Uint64()), nil
}
