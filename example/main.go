package main

import (
	"context"
	"encoding/json"
	"flag"

	"github.com/vocdoni/erc20-storage-proof/ethstorageproof"
	"github.com/vocdoni/erc20-storage-proof/token"
	"gitlab.com/vocdoni/go-dvote/log"

	"github.com/ethereum/go-ethereum/common"
)

const web3 = "https://web3.dappnode.net"

//const holder = "36928500Bc1dCd7af6a2B4008875CC336b927D57"
//const contract = "dac17f958d2ee523a2206206994597c13d831ec7" // tether

func main() {
	logLevel := flag.String("logLevel", "info", "log level")
	contract := flag.String("contract", "", "ERC20 contract address")
	holder := flag.String("holder", "", "address of the token holder")
	flag.Parse()
	log.Init(*logLevel, "stdout")

	ts := token.ERC20Token{}
	ts.Init(context.Background(), web3, *contract)
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
	log.Infof("contract:%s holder:%s balance:%s", *contract, *holder, balance.String())

	slot, amount, err := ts.GetIndexSlot(holderAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("storage data -> slot: %d amount: %s", slot, amount.String())

	sproof, err := ts.GetProof(context.TODO(), holderAddr, nil)
	if err != nil {
		log.Fatal(err)
	}

	sproofBytes, err := json.Marshal(sproof)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("%s", sproofBytes)

	if pv, err := ethstorageproof.VerifyEIP1186(sproof); pv {
		log.Info("account proof is valid!\n")
	} else {
		log.Warnf("account proof is invalid (err %s)\n", err)
	}

}

func trimHex(s string) string {
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	if len(s)&1 == 1 {
		s = "0" + s
	}
	return s
}
