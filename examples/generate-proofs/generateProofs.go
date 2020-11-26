package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/vocdoni/eth-storage-proof/ethstorageproof"
	"github.com/vocdoni/eth-storage-proof/token"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	web3 := flag.String("web3", "https://web3.dappnode.net", "web3 RPC endpoint URL")
	contract := flag.String("contract", "", "ERC20 contract address")
	//holder := flag.String("holder", "", "address of the token holder")
	flag.Parse()

	getProofs(*web3, *contract, erc20Holders)
}

type EthProofs struct {
	BlockNum      *big.Int      `json:"height"`
	IndexSlot     int           `json:"indexSlot"`
	StorageRoot   string        `json:"storageRoot"`
	StorageProofs []HolderProof `json:"storageProofs"`
}

type HolderProof struct {
	Address      string                        `json:"address"`
	StorageProof ethstorageproof.StorageResult `json:"storageProof"`
}

func getProofs(web3, contract string, holders []string) {
	ts := token.ERC20Token{}
	ts.Init(context.Background(), web3, contract)
	slot, _, err := ts.GetIndexSlot(common.HexToAddress(holders[0]))
	if err != nil {
		log.Fatal(err)
	}
	proofs := EthProofs{}
	proofs.IndexSlot = slot

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	sproof, err := ts.GetProof(ctx, common.HexToAddress(holders[0]), nil)
	cancel()
	if err != nil {
		log.Fatalf("Error fetching storageRoot: %v", err)
	}
	proofs.StorageRoot = sproof.StorageHash.Hex()
	blk, err := ts.GetBlock(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	proofs.BlockNum = blk.Number()
	wg := sync.WaitGroup{}
	lock := sync.RWMutex{}
	for _, h := range holders {
		holderAddr := common.HexToAddress(h)
		go func() {
			wg.Add(1)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			sproof, err := ts.GetProof(ctx, holderAddr, nil)
			cancel()
			if err != nil {
				log.Printf("error fetching %s: %s", h, err)
			} else {
				if proofs.StorageRoot != sproof.StorageHash.Hex() {
					log.Printf("wrong storage hash for %s", h)
				} else {
					lock.Lock()
					proofs.StorageProofs = append(proofs.StorageProofs, HolderProof{Address: h, StorageProof: sproof.StorageProof[0]})
					lock.Unlock()
					log.Printf("done for %s", h)
				}
			}
			wg.Done()
		}()
		time.Sleep(time.Millisecond * 10)
	}
	wg.Wait()
	p, err := json.Marshal(proofs)
	if err != nil {
		log.Fatal(err)
	}
	ioutil.WriteFile("proofs.json", p, os.FileMode(0644))
}

var erc20Holders = []string{
	"0x0c5dc58167849ba0c61c952463afb5e3c7970461",
	"0x0c5dc9f198f4bcd221b97e4749f1e06e4b28bff9",
	"0x0c5ddddf9e0a74cd6b23a4be789657f475ebaa9b",
	"0x0c5de5bbdfba99e85572742843e0902fc7863a54",
	"0x0c5dec4d3797f418cef6eccbca62cea7c989b450",
	"0x0c5df0dba295f3ade243df1c63c0262dbc129ecc",
	"0x0c5df4ce909f73a2e04b72e8999b253fdff61e41",
	"0x0c5df4e747521fdcb6ea64bf34c6c4f3d1a8ab51",
	"0x0c5e02ee3b24f1c341254894a87066d5ec4ab6ab",
	"0x0c5e1123e2de815e2f237ae09f33aa0ae7e27be3",
	"0x0c5e19cef0c74435a2ed89d1988b6ff1edf41d46",
	"0x0c5e2c5ff7c66aecc7d38e6a5fb7a82a100c0cf5",
	"0x0c5e2c7f6fc2ee0ebf32dd2ce553a97ded2d1052",
	"0x0c5e2f2c4b8208f4d8af9fb945c79b060babff43",
	"0x0c5e38ea130b70fac691ce13efc758576d1ec97c",
	"0x0c5e3bb1c9355befffbf27e73128c3da98a2d071",
	"0x0c5e4d3039d35659c77c6f6a5b8db50bec3a72dc",
	"0x0c5e50e73e863dbe8fb64d729725486b1b22975b",
	"0x0c5e58129a74df079e9fe557a70d0ee5b5812729",
	"0x0c5e651bd07c039b57903903ed2cbb8f453daea6",
	"0x0c5e713d86456e085944b21dac6dd5816c973015",
	"0x0c60cd9b787d5e9ede1a37522a5cdfd66fe80c26",
	"0x0c60d55a879c7599943ec446ae5264edc0eeb42d",
	"0x0c60d7f02665d58b1c6745570db7c9d740bff766",
	"0x0c60dce470c715cdb18415c3a98ec358085473ac",
	"0x0c60ebc4bee6b9c2e773d389a78f85af889d5dc9",
	"0x0c60f53541e64e45d17a163a2c9bf3a1d90e249e",
	"0x0c60fd1dd94a83475b3ab0d3bd104977210bcbe0",
	"0x0c610431c521febbed671e702c217b3068bb699e",
	"0x0c6118c8c3a74cee7e54cf9283d38b2aa91a3ec9",
	"0x0c611fa42a102f5157b661d10ae2aa1a552148ec",
	"0x0c6128bc1f855b644b28f71d5585425b30fc4c7a",
	"0x0c612d9546025eef3606cbe8242079fa5d0f2f0d",
	"0x0c614a3c037a1966c92c05e0c603f5739e66e613",
	"0x0c614da964ef1b216b3e660b50d714e453b0a026",
	"0x0c6151486345fe62b3d2aa17a31e68930b685d97",
	"0x0c6152caeadb06588108e4179a6505ec8fe85906",
}
