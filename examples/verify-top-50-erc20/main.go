package main

import (
	"context"
	"flag"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	"github.com/vocdoni/storage-proofs-eth-go/token"
	"github.com/vocdoni/storage-proofs-eth-go/token/erc20"
)

const timeout = 60 * time.Second

func main() {
	web3 := flag.String("web3", "https://web3.dappnode.net", "web3 RPC endpoint URL")
	contract := flag.String("contract", "", "ERC20 contract address")
	holder := flag.String("holder", "", "address of the token holder")
	contractType := flag.String("type", "mapbased", "ERC20 contract type (mapbased, minime)")
	flag.Parse()
	var contractAddr common.Address
	if err := contractAddr.UnmarshalText([]byte(*contract)); err != nil {
		log.Fatal(err)
	}
	var holderAddr common.Address
	if err := holderAddr.UnmarshalText([]byte(*holder)); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	rpcCli, err := rpc.DialContext(ctx, *web3)
	if err != nil {
		log.Fatal(err)
	}
	ts, err := erc20.New(ctx, rpcCli, contractAddr)
	if err != nil {
		log.Fatal(err)
	}
	tokenData, err := ts.GetTokenData(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if tokenData.Decimals < 1 {
		log.Fatal("decimals cannot be fetch")
	}

	balance, err := ts.Balance(ctx, holderAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("contract:%v holder:%v balance:%v", contractAddr, holderAddr, balance)
	if balance.Cmp(big.NewRat(0, 1)) == 0 {
		log.Println("no amount for holder")
		return
	}

	var ttype int
	switch *contractType {
	case "mapbased":
		ttype = token.TokenTypeMapbased
	case "minime":
		ttype = token.TokenTypeMinime
	default:
		log.Fatalf("token type not supported %s", *contractType)
	}

	t, err := token.New(ctx, rpcCli, ttype, contractAddr)
	if err != nil {
		log.Fatal(err)
	}
	slot, amount, err := t.DiscoverSlot(ctx, holderAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("storage data -> slot: %d amount: %v", slot, amount)

	blockNumUint64, err := ts.EthCli.BlockNumber(ctx)
	if err != nil {
		log.Fatal(err)
	}
	blockNum := new(big.Int).SetUint64(blockNumUint64)
	sproof, err := t.GetProof(ctx, holderAddr, blockNum, slot)
	if err != nil {
		log.Fatalf("cannot get proof: %v", err)
	}

	if pv, err := ethstorageproof.VerifyEIP1186(sproof); pv {
		log.Printf("account proof is valid!")
	} else {
		log.Printf("account proof is invalid (err %v)", err)
	}
}
