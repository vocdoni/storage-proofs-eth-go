package tokenstate

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"

	contracts "github.com/p4u/erc20-storage-proof/ierc20"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/util"
)

// Web3 holds a reference to a go-ethereum client,
// to an ERC20 like contract and to an ENS.
// It is expected for the ERC20 contract to implement the standard
// optional ERC20 functions: {name, symbol, decimals, totalSupply}
type Web3 struct {
	client    *ethclient.Client
	token     *contracts.TokenCaller
	tokenAddr string
	networkID *big.Int
}

// Init creates and client connection and connects to an ERC20 contract given its address
func (w *Web3) Init(ctx context.Context, web3Endpoint, contractAddress string) error {
	var err error
	// connect to ethereum endpoint
	w.client, err = ethclient.Dial(web3Endpoint)
	if err != nil {
		log.Fatal(err)
	}
	w.networkID, err = w.client.ChainID(ctx)
	if err != nil {
		return err
	}
	log.Debugf("found ethereum network id %s", w.networkID.String())
	// load token contract
	c, err := hex.DecodeString(util.TrimHex(contractAddress))
	if err != nil {
		return err
	}
	caddr := common.Address{}
	caddr.SetBytes(c)
	if w.token, err = contracts.NewTokenCaller(caddr, w.client); err != nil {
		return err
	}

	w.tokenAddr = contractAddress
	log.Infof("loaded token contract %s", caddr.String())
	return nil
}

func (w *Web3) GetTokenData() (*TokenData, error) {
	td := &TokenData{Address: w.tokenAddr}
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

func (w *Web3) Balance(ctx context.Context, address string) (*big.Float, error) {
	b, err := w.token.BalanceOf(&bind.CallOpts{}, common.HexToAddress(address))
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

type TokenData struct {
	Address     string   `json:"address"`
	Name        string   `json:"name"`
	Symbol      string   `json:"symbol"`
	Decimals    uint8    `json:"decimals"`
	TotalSupply *big.Int `json:"totalSupply,omitempty"`
}

func (t *TokenData) String() string {
	return fmt.Sprintf(`{"name":%s,"symbol":%s,"decimals":%s,"totalSupply":%s}`, t.Name, t.Symbol, string(t.Decimals), t.TotalSupply.String())
}

// TokenName wraps the name() function contract call
func (w *Web3) TokenName() (string, error) {
	return w.token.Name(nil)
}

// TokenSymbol wraps the symbol() function contract call
func (w *Web3) TokenSymbol() (string, error) {
	return w.token.Symbol(nil)
}

// TokenDecimals wraps the decimals() function contract call
func (w *Web3) TokenDecimals() (uint8, error) {
	return w.token.Decimals(nil)
}

// TokenTotalSupply wraps the totalSupply function contract call
func (w *Web3) TokenTotalSupply() (*big.Int, error) {
	return w.token.TotalSupply(nil)
}
