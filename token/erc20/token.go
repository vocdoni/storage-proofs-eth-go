package erc20

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type TokenData struct {
	Address     common.Address `json:"address"`
	Name        string         `json:"name"`
	Symbol      string         `json:"symbol"`
	Decimals    uint8          `json:"decimals"`
	TotalSupply *big.Int       `json:"totalSupply,omitempty"`
}

func (t *TokenData) String() string {
	return fmt.Sprintf(`{"name":%s,"symbol":%s,"decimals":%v,"totalSupply":%v}`,
		t.Name, t.Symbol, t.Decimals, t.TotalSupply)
}
