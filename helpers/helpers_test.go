package helpers

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	qt "github.com/frankban/quicktest"
)

func TestValueToBalance(t *testing.T) {
	c := qt.New(t)
	type data struct {
		inputValue     string
		inputDecimals  int
		outputBalance  string
		outputIBalance string
	}
	vectors := []data{{
		inputValue:     "0x00000000000293fb5ca8d27b5662e57700000000000000000000000000c304f2",
		inputDecimals:  18,
		outputBalance:  "1060549995705646568037077887575325019587292552.758520904839005426",
		outputIBalance: "1060549995705646568037077887575325019587292552758520904839005426",
	}, {
		inputValue:     "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		inputDecimals:  18,
		outputBalance:  "115792089237316195423570985008687907853269984665640564039457.584007913129639935",
		outputIBalance: "115792089237316195423570985008687907853269984665640564039457584007913129639935",
	}}
	for i, v := range vectors {
		value := hexutil.MustDecode(v.inputValue)
		balance, ibalance, err := ValueToBalance(value, v.inputDecimals)
		c.Run(fmt.Sprintf("i=%v", i), func(c *qt.C) {
			c.Assert(err, qt.IsNil)
			c.Check(balance.FloatString(v.inputDecimals), qt.Equals, v.outputBalance)
			c.Check(ibalance.String(), qt.Equals, v.outputIBalance)
		})
	}
}
