package erc20

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	"github.com/vocdoni/storage-proofs-eth-go/helpers"
	contracts "github.com/vocdoni/storage-proofs-eth-go/ierc20"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// ERC20Token holds a reference to a go-ethereum client,
// to an ERC20 like contract and to an ENS.
// It is expected for the ERC20 contract to implement the standard
// optional ERC20 functions: {name, symbol, decimals, totalSupply}
type ERC20Token struct {
	RPCcli    *rpc.Client
	Ethcli    *ethclient.Client
	token     *contracts.TokenCaller
	TokenAddr common.Address
	networkID *big.Int
}

// Init creates and client connection and connects to an ERC20 contract given its address
func (w *ERC20Token) Init(ctx context.Context, web3Endpoint string,
	contractAddress common.Address) error {
	var err error
	// if web3Endpoint is empty assume the client already exists
	if web3Endpoint != "" {
		// connect to ethereum endpoint if required
		w.RPCcli, err = rpc.Dial(web3Endpoint)
		if err != nil {
			return err
		}
		w.Ethcli = ethclient.NewClient(w.RPCcli)
	} else {
		if w.RPCcli == nil {
			return fmt.Errorf("RPC node client is not set")
		}
		if w.Ethcli == nil {
			//lint:ignore ST1005 "Ethereum" is a proper noun, so it's capitalized
			return fmt.Errorf("Ethereum client is not set")
		}
	}
	w.networkID, err = w.Ethcli.ChainID(ctx)
	if err != nil {
		return err
	}
	// load token contract
	w.TokenAddr = contractAddress
	if w.token, err = contracts.NewTokenCaller(w.TokenAddr, w.Ethcli); err != nil {
		return err
	}

	return nil
}

// GetTokenData gets useful data abount the token
func (w *ERC20Token) GetTokenData(ctx context.Context) (*TokenData, error) {
	td := &TokenData{Address: w.TokenAddr}
	var err error

	if td.Name, err = w.TokenName(ctx); err != nil {
		if strings.Contains(err.Error(), "unmarshal an empty string") {
			td.Name = "unknown-name"
		} else {
			return nil, fmt.Errorf("unable to get token name data: %s", err)
		}
	}

	if td.Symbol, err = w.TokenSymbol(ctx); err != nil {
		if strings.Contains(err.Error(), "unmarshal an empty string") {
			td.Symbol = "unknown-symbol"
		} else {
			return nil, fmt.Errorf("unable to get token symbol data: %s", err)
		}
	}

	if td.Decimals, err = w.TokenDecimals(ctx); err != nil {
		return nil, fmt.Errorf("unable to get token decimals data: %s", err)
	}

	if td.TotalSupply, err = w.TokenTotalSupply(ctx); err != nil {
		return nil, fmt.Errorf("unable to get token supply data: %s", err)
	}

	return td, nil
}

// Balance returns the current address balance
func (w *ERC20Token) Balance(ctx context.Context, address common.Address) (*big.Rat, error) {
	b, err := w.token.BalanceOf(&bind.CallOpts{Context: ctx}, address)
	if err != nil {
		return nil, err
	}
	decimals, err := w.TokenDecimals(ctx)
	if err != nil {
		return nil, err
	}
	return helpers.BalanceToRat(b, int(decimals)), nil
}

// TokenName wraps the name() function contract call
func (w *ERC20Token) TokenName(ctx context.Context) (string, error) {
	return w.token.Name(&bind.CallOpts{Context: ctx})
}

// TokenSymbol wraps the symbol() function contract call
func (w *ERC20Token) TokenSymbol(ctx context.Context) (string, error) {
	return w.token.Symbol(&bind.CallOpts{Context: ctx})
}

// TokenDecimals wraps the decimals() function contract call
func (w *ERC20Token) TokenDecimals(ctx context.Context) (uint8, error) {
	return w.token.Decimals(&bind.CallOpts{Context: ctx})
}

// TokenTotalSupply wraps the totalSupply function contract call
func (w *ERC20Token) TokenTotalSupply(ctx context.Context) (*big.Int, error) {
	return w.token.TotalSupply(&bind.CallOpts{Context: ctx})
}

// GetProof calls the eth_getProof web3 method.  If block is nil, the proof at
// the latest block will be retreived.
func (w *ERC20Token) GetProof(ctx context.Context, keys [][]byte,
	block *big.Int) (*ethstorageproof.StorageProof, error) {
	blockData, err := w.GetBlock(ctx, block)
	if err != nil {
		return nil, err
	}
	var resp ethstorageproof.StorageProof
	if err := w.RPCcli.CallContext(
		ctx,
		&resp,
		"eth_getProof",
		fmt.Sprintf("0x%x", w.TokenAddr),
		ethstorageproof.SliceData(keys),
		helpers.ToBlockNumArg(block),
	); err != nil {
		return nil, err
	}
	resp.StateRoot = blockData.Root()
	resp.Height = blockData.Header().Number
	return &resp, nil
}

// GetBlock gets an Ethereum block given its height
func (w *ERC20Token) GetBlock(ctx context.Context, number *big.Int) (*types.Block, error) {
	return w.Ethcli.BlockByNumber(ctx, number)
}
