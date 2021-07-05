package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	"github.com/vocdoni/storage-proofs-eth-go/helpers"
	"github.com/vocdoni/storage-proofs-eth-go/token"
	"github.com/vocdoni/storage-proofs-eth-go/token/erc20"
	"github.com/vocdoni/storage-proofs-eth-go/token/minime"
)

func main() {
	web3 := flag.String("web3", "https://web3.dappnode.net", "web3 RPC endpoint URL")
	contract := flag.String("contract", "", "ERC20 contract address")
	holder := flag.String("holder", "", "address of the token holder")
	contractType := flag.String("type", "mapbased", "ERC20 contract type (mapbased, minime)")
	height := flag.Int64("height", 0, "ethereum height (0 becomes last block)")
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

	var ttype int
	switch *contractType {
	case "mapbased":
		ttype = token.TokenTypeMapbased
	case "minime":
		ttype = token.TokenTypeMinime
	default:
		log.Fatalf("token type not supported %s", *contractType)
	}

	t, err := token.NewToken(ttype, *contract, *web3)
	if err != nil {
		log.Fatal(err)
	}
	slot, amount, err := t.DiscoverSlot(common.HexToAddress(*holder))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("storage data -> slot: %d amount: %s", slot, amount.String())

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

	// DBG BEGIN
	for _, proof := range sproof.StorageProof[:1] {
		proofJSON, _ := json.MarshalIndent(proof, "", "  ")
		fmt.Printf("DBG\n%v\n", string(proofJSON))

		proofDB := ethstorageproof.NewMemDB()
		for _, node := range proof.Proof {
			// value, err := hexutil.Decode(node)
			// if err != nil {
			// 	log.Fatal(err)
			// }
			key := crypto.Keccak256(node)
			// fmt.Printf("%v -> %v\n", hexutil.Encode(key), hexutil.Encode(value))
			// var decValue interface{}
			// err = rlp.DecodeBytes(value, &decValue)
			// if err != nil {
			// 	log.Fatal(err)
			// }
			// for _, v := range decValue.([]interface{}) {
			// 	fmt.Printf("> %v\n", hexutil.Encode(v.([]byte)))
			// }
			proofDB.Put(key, node)
		}
		// key, err := hexutil.Decode(fmt.Sprintf("0x%s", proof.Key))
		// if err != nil {
		// 	log.Fatal(err)
		// }
		path := crypto.Keccak256(proof.Key)
		fmt.Printf("DBG key: %v\n", hexutil.Encode(path))
		fmt.Printf("WantHash: %v\n", sproof.StorageHash)

		res, err := trie.VerifyProof(sproof.StorageHash, path, proofDB)
		fmt.Printf("VerifyProof: %v, %v\n", res, err)
	}
	// DBG END
	return

	switch ttype {
	case token.TokenTypeMinime:
		balance, fullBalance, block, err := minime.ParseMinimeValue(
			sproof.StorageProof[0].Value.String(),
			int(tokenData.Decimals),
		)
		if err != nil {
			log.Printf("warning: %v", err)
		}
		log.Printf("balance on block %s: %s", block.String(), balance.String())
		log.Printf("hex balance: %x\n", fullBalance.Bytes())
		log.Printf("storage root: %x\n", sproof.StorageHash)

		sproofBytes, err := json.MarshalIndent(sproof.StorageProof, "", " ")
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s\n", sproofBytes)

		if err := minime.VerifyProof(
			common.HexToAddress(*holder),
			sproof.StorageHash,
			sproof.StorageProof,
			slot,
			fullBalance,
			block,
		); err != nil {
			log.Fatal(err)
		}
	case token.TokenTypeMapbased:
		balance, err := helpers.ValueToBalance(
			sproof.StorageProof[0].Value.String(),
			int(tokenData.Decimals),
		)
		if err != nil {
			log.Printf("warning: %v", err)
		}
		log.Printf("Mapbased balance on block %s: %s", block.Number().String(), balance.String())
	}

	if pv, err := ethstorageproof.VerifyEIP1186(sproof); pv {
		log.Printf("account proof is valid!")
	} else {
		log.Printf("account proof is invalid: (%v)", err)
	}
}
