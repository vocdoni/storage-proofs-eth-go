package main

import (
	"context"
	"flag"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	"github.com/vocdoni/storage-proofs-eth-go/token"
	"github.com/vocdoni/storage-proofs-eth-go/token/erc20"
)

func main() {
	web3 := flag.String("web3", "https://web3.dappnode.net", "web3 RPC endpoint URL")
	contract := flag.String("contract", "", "ERC20 contract address")
	holder := flag.String("holder", "", "address of the token holder")
	flag.Parse()

	ts := erc20.ERC20Token{}
	ts.Init(context.Background(), *web3, *contract)
	tokenData, err := ts.GetTokenData()
	if err != nil {
		log.Fatal(err)
	}
	if tokenData.Decimals < 1 {
		log.Fatal("decimals cannot be fetch")
	}
	holderAddr := common.HexToAddress(*holder)

	balance, err := ts.Balance(holderAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("contract:%s holder:%s balance:%s", *contract, *holder, balance.String())
	if a, _ := balance.Uint64(); a == 0 {
		log.Println("no amount for holder")
		return
	}

	t, err := token.NewToken(token.TokenTypeMapbased, *contract, *web3)
	if err != nil {
		log.Fatal(err)
	}
	slot, amount, err := t.DiscoverSlot(common.HexToAddress(*holder))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("storage data -> slot: %d amount: %s", slot, amount.String())

	block, err := ts.GetBlock(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	sproof, err := t.GetProof(holderAddr, block.Number(), slot)
	if err != nil {
		log.Fatalf("cannot get proof: %v", err)
	}
	/*	sproofBytes, err := json.MarshalIndent(sproof.StorageProof, "", " ")
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s\n", sproofBytes)
	*/
	if pv, err := ethstorageproof.VerifyEIP1186(sproof); pv {
		log.Printf("account proof is valid!")
	} else {
		log.Printf("account proof is invalid (err %v)", err)
	}
}
