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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	"github.com/vocdoni/storage-proofs-eth-go/token"
)

const timeout = 60 * time.Second

func main() {
	web3 := flag.String("web3", "https://web3.dappnode.net", "web3 RPC endpoint URL")
	contract := flag.String("contract", "", "ERC20 contract address")
	holderFile := flag.String("holderFile", "", "text file with holder addresses (separated by line)")
	flag.Parse()
	var contractAddr common.Address
	if err := contractAddr.UnmarshalText([]byte(*contract)); err != nil {
		log.Fatal(err)
	}
	data, err := ioutil.ReadFile(*holderFile)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	rpcCli, err := rpc.DialContext(ctx, *web3)
	if err != nil {
		log.Fatal(err)
	}
	getProofs(ctx, rpcCli, contractAddr, strings.Split(string(data), "\n"))
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

func getProofs(ctx context.Context, rpcCli *rpc.Client, contract common.Address, holders []string) {
	t, err := token.New(ctx, rpcCli, token.TokenTypeMapbased, contract)
	if err != nil {
		log.Fatal(err)
	}
	slot, _, err := t.DiscoverSlot(ctx, common.HexToAddress(holders[0]))
	if err != nil {
		log.Fatal(err)
	}
	proofs := EthProofs{}
	proofs.IndexSlot = slot

	sproof, err := t.GetProof(ctx, common.HexToAddress(holders[0]), nil, slot)
	if err != nil {
		log.Fatalf("Error fetching storageRoot: %v", err)
	}
	proofs.StorageRoot = sproof.StorageHash.Hex()
	blockNumUint64, err := ethclient.NewClient(rpcCli).BlockNumber(ctx)
	if err != nil {
		log.Fatal(err)
	}
	proofs.BlockNum = new(big.Int).SetUint64(blockNumUint64)
	wg := sync.WaitGroup{}
	lock := sync.RWMutex{}
	for _, h := range holders {
		if len(h) < 20 {
			continue
		}
		wg.Add(1)
		go func() {
			holderAddr := common.HexToAddress(h)
			sproof, err := t.GetProof(ctx, holderAddr, proofs.BlockNum, slot)
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
	ioutil.WriteFile("proofs.json", p, os.FileMode(0o644))
}
