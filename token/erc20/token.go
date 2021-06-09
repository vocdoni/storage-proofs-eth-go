package erc20

import (
	"fmt"
	"math/big"
)

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
