package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const web3 = "https://mainnet.infura.io/v3/63831ad4d7034546a84339d02cba2eab"

//const holder = "36928500Bc1dCd7af6a2B4008875CC336b927D57"
//const contract = "dac17f958d2ee523a2206206994597c13d831ec7" // tether
//const decimals = 6

func main() {
	contract := flag.String("contract", "", "ERC20 contract address")
	holder := flag.String("holder", "", "address of the token holder")
	decimals := flag.Int("decimals", 18, "number of decimals for the ERC20 contract")
	flag.Parse()
	*contract = strings.TrimPrefix(*contract, "0x")
	*holder = strings.TrimPrefix(*holder, "0x")

	contractb, err := hex.DecodeString(*contract)
	if err != nil {
		panic(err)
	}
	c, err := ethclient.Dial(web3)
	if err != nil {
		panic(err)
	}
	addr := common.Address{}
	copy(addr[:], contractb[:20])

	var slot [32]byte
	for i := 0; i < 10; i++ {
		slot = getSlot(*holder, i)
		//fmt.Printf("querying for contract %x and slot [%d] %x\n", addr, i, slot)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		value, err := c.StorageAt(ctx, addr, slot, nil)
		if err != nil {
			panic(err)
		}
		cancel()

		// Parse balance value
		value = common.TrimLeftZeroes(value)
		amount := big.NewFloat(0)
		if _, ok := amount.SetString(fmt.Sprintf("0x%x", value)); !ok {
			continue
		}
		amount.Mul(amount, big.NewFloat(1/(math.Pow10(*decimals))))

		fmt.Printf("Found balance on slot index %d: %s\n", i, amount.String())
	}
}

func getSlot(holder string, position int) (slot [32]byte) {
	hl, err := hex.DecodeString(holder)
	if err != nil {
		panic(err)
	}
	hl = common.LeftPadBytes(hl, 32)
	posHex := fmt.Sprintf("%x", position)
	if len(posHex)%2 == 1 {
		posHex = "0" + posHex
	}
	p, err := hex.DecodeString(posHex)
	if err != nil {
		panic(err)
	}
	p = common.LeftPadBytes(p, 32)

	hash := crypto.Keccak256(hl, p)
	copy(slot[:], hash[:32])
	return
}
