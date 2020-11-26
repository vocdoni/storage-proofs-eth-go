package main

import (
	"context"
	"flag"
	"log"

	"github.com/vocdoni/eth-storage-proof/ethstorageproof"
	"github.com/vocdoni/eth-storage-proof/token"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	web3 := flag.String("web3", "https://web3.dappnode.net", "web3 RPC endpoint URL")
	contract := flag.String("contract", "", "ERC20 contract address")
	holder := flag.String("holder", "", "address of the token holder")
	flag.Parse()
	ts := token.ERC20Token{}
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

	slot, amount, err := ts.GetIndexSlot(holderAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("storage data -> slot: %d amount: %s", slot, amount.String())

	sproof, err := ts.GetProof(context.TODO(), holderAddr, nil)
	if err != nil {
		log.Fatal(err)
	}

	//sproofBytes, err := json.Marshal(sproof)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//log.Print("%s", sproofBytes)

	if pv, err := ethstorageproof.VerifyEIP1186(sproof); pv {
		log.Printf("account proof is valid!")
	} else {
		log.Printf("account proof is invalid (err %s)", err)
	}

}
