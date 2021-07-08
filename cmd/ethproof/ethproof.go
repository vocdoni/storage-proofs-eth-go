package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/vocdoni/storage-proofs-eth-go/helpers"
	"github.com/vocdoni/storage-proofs-eth-go/token"
	"github.com/vocdoni/storage-proofs-eth-go/token/erc20"
	"github.com/vocdoni/storage-proofs-eth-go/token/mapbased"
	"github.com/vocdoni/storage-proofs-eth-go/token/minime"
)

func main() {
	web3 := flag.String("web3", "https://web3.dappnode.net", "web3 RPC endpoint URL")
	contract := flag.String("contract", "", "ERC20 contract address")
	holder := flag.String("holder", "", "address of the token holder")
	contractType := flag.String("type", "mapbased", "ERC20 contract type (mapbased, minime)")
	height := flag.Int64("height", 0, "ethereum height (0 becomes last block)")
	flag.Parse()

	var contractAddr common.Address
	if err := contractAddr.UnmarshalText([]byte(*contract)); err != nil {
		log.Fatal(err)
	}
	var holderAddr common.Address
	if err := holderAddr.UnmarshalText([]byte(*holder)); err != nil {
		log.Fatal(err)
	}

	ts := erc20.ERC20Token{}
	if err := ts.Init(context.Background(), *web3, contractAddr); err != nil {
		log.Fatal(err)
	}
	tokenData, err := ts.GetTokenData()
	if err != nil {
		log.Fatal(err)
	}
	if tokenData.Decimals < 1 {
		log.Fatal("decimals cannot be fetch")
	}
	decimals := int(tokenData.Decimals)

	balance, err := ts.Balance(holderAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("contract:%v holder:%v balance:%s", contractAddr, holderAddr,
		balance.FloatString(decimals))
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

	t, err := token.NewToken(ttype, contractAddr, *web3)
	if err != nil {
		log.Fatal(err)
	}
	slot, amount, err := t.DiscoverSlot(holderAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("storage data -> slot: %d amount: %s", slot, amount.FloatString(decimals))

	var blockNum *big.Int
	if *height > 0 {
		blockNum = new(big.Int).SetInt64(*height)
	}
	block, err := ts.GetBlock(context.Background(), blockNum)
	if err != nil {
		log.Fatal(err)
	}
	sproof, err := t.GetProof(holderAddr, block.Number(), slot)
	if err != nil {
		log.Fatalf("cannot get proof: %v", err)
	}

	switch ttype {
	case token.TokenTypeMinime:
		balance, fullBalance, block := minime.ParseMinimeValue(
			sproof.StorageProof[0].Value,
			int(tokenData.Decimals),
		)
		log.Printf("balance on block %s: %s", block.String(), balance.FloatString(decimals))
		log.Printf("hex balance: %x\n", fullBalance.Bytes())
		log.Printf("storage root: %x\n", sproof.StorageHash)
		if err := minime.VerifyProof(
			holderAddr,
			sproof.StorageHash,
			sproof.StorageProof,
			slot,
			fullBalance,
			block,
		); err != nil {
			log.Fatal(err)
		}
	case token.TokenTypeMapbased:
		balance, fullBalance := helpers.ValueToBalance(
			sproof.StorageProof[0].Value,
			int(tokenData.Decimals),
		)
		log.Printf("mapbased balance on block %s: %s", block.Number().String(),
			balance.FloatString(decimals))
		if err := mapbased.VerifyProof(
			holderAddr,
			sproof.StorageHash,
			sproof.StorageProof[0],
			slot,
			fullBalance,
			nil,
		); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("token type not supported")
	}

	sproofBytes, err := json.MarshalIndent(sproof, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s\n", sproofBytes)
	log.Println("proof is valid!")
}
