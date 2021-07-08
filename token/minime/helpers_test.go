package minime

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	qt "github.com/frankban/quicktest"
)

func TestParseMinimeValue(t *testing.T) {
	c := qt.New(t)
	type data struct {
		inputValue     string
		inputDecimals  int
		outputBalance  string
		outputIBalance string
		outputBlock    string
	}
	vectors := []data{{
		inputValue:     "0x00000000000293fb5ca8d27b5662e57700000000000000000000000000c304f2",
		inputDecimals:  18,
		outputBalance:  "3116676.321791472042173815",
		outputIBalance: "3116676321791472042173815",
		outputBlock:    "12780786",
	}, {
		inputValue:     "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		inputDecimals:  18,
		outputBalance:  "340282366920938463463.374607431768211455",
		outputIBalance: "340282366920938463463374607431768211455",
		outputBlock:    "340282366920938463463374607431768211455",
	}}
	for i, v := range vectors {
		value := hexutil.MustDecode(v.inputValue)
		balance, ibalance, mblock, err := ParseMinimeValue(value, v.inputDecimals)
		c.Run(fmt.Sprintf("i=%v", i), func(c *qt.C) {
			c.Assert(err, qt.IsNil)
			c.Check(balance.FloatString(v.inputDecimals), qt.Equals, v.outputBalance)
			c.Check(ibalance.String(), qt.Equals, v.outputIBalance)
			c.Check(mblock.String(), qt.Equals, v.outputBlock)
		})
	}
}
