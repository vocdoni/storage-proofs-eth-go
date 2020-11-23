package token

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/vocdoni/eth-storage-proof/ethstorageproof"
	contracts "github.com/vocdoni/eth-storage-proof/ierc20"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var ErrSlotNotFound = errors.New("storage slot not found")

// ERC20Token holds a reference to a go-ethereum client,
// to an ERC20 like contract and to an ENS.
// It is expected for the ERC20 contract to implement the standard
// optional ERC20 functions: {name, symbol, decimals, totalSupply}
type ERC20Token struct {
	rpccli    *rpc.Client
	ethcli    *ethclient.Client
	token     *contracts.TokenCaller
	tokenAddr []byte
	networkID *big.Int
}

// Init creates and client connection and connects to an ERC20 contract given its address
func (w *ERC20Token) Init(ctx context.Context, web3Endpoint, contractAddress string) error {
	var err error
	// connect to ethereum endpoint
	w.rpccli, err = rpc.Dial(web3Endpoint)
	if err != nil {
		return err
	}
	w.ethcli = ethclient.NewClient(w.rpccli)

	w.networkID, err = w.ethcli.ChainID(ctx)
	if err != nil {
		return err
	}
	// load token contract
	w.tokenAddr, err = hex.DecodeString(trimHex(contractAddress))
	if err != nil {
		return err
	}
	caddr := common.Address{}
	caddr.SetBytes(w.tokenAddr)
	if w.token, err = contracts.NewTokenCaller(caddr, w.ethcli); err != nil {
		return err
	}

	return nil
}

func (w *ERC20Token) GetTokenData() (*TokenData, error) {
	td := &TokenData{Address: fmt.Sprintf("%x", w.tokenAddr)}
	var err error

	if td.Name, err = w.TokenName(); err != nil {
		return nil, fmt.Errorf("unable to get token data: %s", err)
	}

	if td.Symbol, err = w.TokenSymbol(); err != nil {
		return nil, fmt.Errorf("unable to get token data: %s", err)
	}

	if td.Decimals, err = w.TokenDecimals(); err != nil {
		return nil, fmt.Errorf("unable to get token data: %s", err)
	}

	if td.TotalSupply, err = w.TokenTotalSupply(); err != nil {
		return nil, fmt.Errorf("unable to get token data: %s", err)
	}

	return td, nil
}

// Balance returns the current address balance
func (w *ERC20Token) Balance(address common.Address) (*big.Float, error) {
	b, err := w.token.BalanceOf(&bind.CallOpts{}, address)
	if err != nil {
		return nil, err
	}
	decimals, err := w.TokenDecimals()
	if err != nil {
		return nil, err
	}
	f := big.NewFloat(float64(0))
	f.SetString(b.String())
	f.Mul(f, big.NewFloat(1/(math.Pow10(int(decimals)))))
	return f, nil
}

// TokenName wraps the name() function contract call
func (w *ERC20Token) TokenName() (string, error) {
	return w.token.Name(nil)
}

// TokenSymbol wraps the symbol() function contract call
func (w *ERC20Token) TokenSymbol() (string, error) {
	return w.token.Symbol(nil)
}

// TokenDecimals wraps the decimals() function contract call
func (w *ERC20Token) TokenDecimals() (uint8, error) {
	return w.token.Decimals(nil)
}

// TokenTotalSupply wraps the totalSupply function contract call
func (w *ERC20Token) TokenTotalSupply() (*big.Int, error) {
	return w.token.TotalSupply(nil)
}

// GetProof returns the storage merkle proofs for the acount holder
func (w *ERC20Token) GetProof(ctx context.Context, holder common.Address, block *types.Block) (*ethstorageproof.StorageProof, error) {
	islot, _, err := w.GetIndexSlot(holder)
	if err != nil {
		return nil, err
	}
	slot, err := GetSlot(holder.Hex(), islot)
	if err != nil {
		return nil, err
	}
	keys := []string{fmt.Sprintf("%x", slot)}
	if block == nil {
		block, err = w.getBlock(ctx, nil)
		if err != nil {
			return nil, err
		}
	}
	return w.getProof(ctx, keys, block)
}

func (w *ERC20Token) getProof(ctx context.Context, keys []string, block *types.Block) (*ethstorageproof.StorageProof, error) {
	var resp *ethstorageproof.StorageProof
	err := w.rpccli.CallContext(ctx, &resp, "eth_getProof", fmt.Sprintf("0x%x", w.tokenAddr), keys, toBlockNumArg(block.Number()))
	resp.Height = block.Header().Number
	resp.StateRoot = block.Root()
	return resp, err
}

// GetIndexSlot tries to find the EVM storage index slot.
// A token holder address must be provided in order to have a balance to search and compare.
// Returns ErrSlotNotFound if the slot cannot be found.
// If found, returns also the amount stored.
func (w *ERC20Token) GetIndexSlot(holder common.Address) (int, *big.Float, error) {
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
		slot, err = GetSlot(holder.Hex(), i)
		if err != nil {
			return index, nil, fmt.Errorf("GetSlot: %w", err)
		}
		// Get Storage
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		value, err := w.ethcli.StorageAt(ctx, addr, slot, nil)
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

func (w *ERC20Token) getBlock(ctx context.Context, number *big.Int) (*types.Block, error) {
	return w.ethcli.BlockByNumber(ctx, number)
}
