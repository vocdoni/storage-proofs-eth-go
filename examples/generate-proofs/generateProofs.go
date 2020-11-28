package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vocdoni/eth-storage-proof/ethstorageproof"
	"github.com/vocdoni/eth-storage-proof/token"

	"github.com/ethereum/go-ethereum/common"
)

func main() {
	web3 := flag.String("web3", "https://web3.dappnode.net", "web3 RPC endpoint URL")
	contract := flag.String("contract", "", "ERC20 contract address")
	holderFile := flag.String("holderFile", "", "text file with holder addresses (separated by line)")
	flag.Parse()
	data, err := ioutil.ReadFile(*holderFile)
	if err != nil {
		log.Fatal(err)
	}
	getProofs(*web3, *contract, strings.Split(string(data), "\n"))
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
		if len(h) < 20 {
			continue
		}
		go func() {
			wg.Add(1)
			holderAddr := common.HexToAddress(h)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			sproof, err := ts.GetProof(ctx, holderAddr, nil)
			cancel()
			if err != nil {
				log.Printf("error fetching %s: %v", holderAddr.Hex(), err)
			} else {
				if proofs.StorageRoot != sproof.StorageHash.Hex() {
					log.Printf("wrong storage hash for %s", holderAddr.Hex())
				} else {
					lock.Lock()
					proofs.StorageProofs = append(proofs.StorageProofs, HolderProof{Address: holderAddr.Hex(), StorageProof: sproof.StorageProof[0]})
					lock.Unlock()
					log.Printf("done for %s", holderAddr.Hex())
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
