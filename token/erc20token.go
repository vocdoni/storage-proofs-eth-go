package token

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	contracts "github.com/vocdoni/storage-proofs-eth-go/ierc20"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// ErrSlotNotFound represents the storage slot not found error
var ErrSlotNotFound = errors.New("storage slot not found")

// ERC20Token holds a reference to a go-ethereum client,
// to an ERC20 like contract and to an ENS.
// It is expected for the ERC20 contract to implement the standard
// optional ERC20 functions: {name, symbol, decimals, totalSupply}
type ERC20Token struct {
	RPCcli    *rpc.Client
	Ethcli    *ethclient.Client
	token     *contracts.TokenCaller
	tokenAddr []byte
	networkID *big.Int
}

// Init creates and client connection and connects to an ERC20 contract given its address
func (w *ERC20Token) Init(ctx context.Context, web3Endpoint, contractAddress string) error {
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
			return fmt.Errorf("Ethereum client is not set")
		}
	}
	w.networkID, err = w.Ethcli.ChainID(ctx)
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
	if w.token, err = contracts.NewTokenCaller(caddr, w.Ethcli); err != nil {
		return err
	}

	return nil
}

// GetTokenData gets useful data abount the token
func (w *ERC20Token) GetTokenData() (*TokenData, error) {
	td := &TokenData{Address: fmt.Sprintf("%x", w.tokenAddr)}
	var err error

	if td.Name, err = w.TokenName(); err != nil {
		if strings.Contains(err.Error(), "unmarshal an empty string") {
			td.Name = "unknown-name"
		} else {
			return nil, fmt.Errorf("unable to get token name data: %s", err)
		}
	}

	if td.Symbol, err = w.TokenSymbol(); err != nil {
		if strings.Contains(err.Error(), "unmarshal an empty string") {
			td.Symbol = "unknown-symbol"
		} else {
			return nil, fmt.Errorf("unable to get token symbol data: %s", err)
		}
	}

	if td.Decimals, err = w.TokenDecimals(); err != nil {
		return nil, fmt.Errorf("unable to get token decimals data: %s", err)
	}

	if td.TotalSupply, err = w.TokenTotalSupply(); err != nil {
		return nil, fmt.Errorf("unable to get token supply data: %s", err)
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

func (w *ERC20Token) getProof(ctx context.Context, keys []string, block *types.Block) (*ethstorageproof.StorageProof, error) {
	if block == nil {
		return nil, fmt.Errorf("block is nil")
	}
	var resp ethstorageproof.StorageProof
	err := w.RPCcli.CallContext(ctx, &resp, "eth_getProof", fmt.Sprintf("0x%x", w.tokenAddr), keys, toBlockNumArg(block.Number()))
	if err != nil {
		return nil, err
	}
	resp.StateRoot = block.Root()
	resp.Height = block.Header().Number
	return &resp, err
}

// GetBlock gets an Ethereum block given its height
func (w *ERC20Token) GetBlock(ctx context.Context, number *big.Int) (*types.Block, error) {
	return w.Ethcli.BlockByNumber(ctx, number)
}
