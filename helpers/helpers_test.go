package helpers

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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
		balance, ibalance := ValueToBalance(value, v.inputDecimals)
		c.Run(fmt.Sprintf("i=%v", i), func(c *qt.C) {
			c.Check(balance.FloatString(v.inputDecimals), qt.Equals, v.outputBalance)
			c.Check(ibalance.String(), qt.Equals, v.outputIBalance)
		})
	}
}

func TestGetMapSlot(t *testing.T) {
	c := qt.New(t)

	address := common.HexToAddress("0xbd9c69654b8f3e5978dfd138b00cb0be29f28ccf")
	position := 1
	mapSlot := GetMapSlot(address, position)
	c.Check(common.Hash(mapSlot).Hex(), qt.Equals,
		"0x4a985c9a291a06b2854315c3a75ca2c1065ef62e859e2534b655d306748c16d4")
}

func TestGetArraySlot(t *testing.T) {
	c := qt.New(t)

	arraySlot := GetArraySlot(3)
	c.Check(common.Hash(arraySlot).Hex(), qt.Equals,
		"0xc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b")
}
