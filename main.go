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

	"github.com/p4u/erc20-storage-proof/proof"
	"github.com/p4u/erc20-storage-proof/tokenstate"
	"gitlab.com/vocdoni/go-dvote/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const web3 = "https://mainnet.infura.io/v3/63831ad4d7034546a84339d02cba2eab"

//const holder = "36928500Bc1dCd7af6a2B4008875CC336b927D57"
//const contract = "dac17f958d2ee523a2206206994597c13d831ec7" // tether
//const decimals = 6

func main() {
	logLevel := flag.String("logLevel", "info", "log level")
	contract := flag.String("contract", "", "ERC20 contract address")
	holder := flag.String("holder", "", "address of the token holder")
	//	decimals := flag.Int("decimals", 18, "number of decimals for the ERC20 contract")
	flag.Parse()
	log.Init(*logLevel, "stdout")

	ts := tokenstate.Web3{}
	ts.Init(context.Background(), web3, *contract)
	tokenData, err := ts.GetTokenData()
	if err != nil {
		log.Fatal(err)
	}
	if tokenData.Decimals < 1 {
		log.Fatal("decimals cannot be fetch")
	}
	balance, err := ts.Balance(context.TODO(), *holder)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("contract:%s holder:%s balance:%s", *contract, *holder, balance.String())

	*contract = strings.TrimPrefix(*contract, "0x")
	*holder = strings.TrimPrefix(*holder, "0x")
	contractb, err := hex.DecodeString(*contract)
	if err != nil {
		log.Fatal(err)
	}
	c, err := ethclient.Dial(web3)
	if err != nil {
		log.Fatal(err)
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
			log.Error(err)
			continue
		}
		cancel()

		/*		pr, err := json.Marshal(proof)
				if err != nil {
					log.Error(err)
					continue
				}
				log.Infof("Proof: %s", pr)
		*/
		/*
			db := trie.NewDatabase(memorydb.New())
			trie.New(common.Hash{}, db)
			mdb := memorydb.New()
			mdb.Put(proof.StorageHash.Bytes(),proof.StorageProof[0].Proof[0])

			for _, v := range proof.StorageProof[0].Proof {
			mdb.Put()
			}
			trie.VerifyProof(proof.StorageHash, nil, mdb)
		*/

		// Parse balance value
		value = common.TrimLeftZeroes(value)
		amount := big.NewFloat(0)
		if _, ok := amount.SetString(fmt.Sprintf("0x%x", value)); !ok {
			continue
		}
		amount.Mul(amount, big.NewFloat(1/(math.Pow10(int(tokenData.Decimals)))))

		log.Infof("found balance on slot index %d: %s", i, amount.String())
		if amount.Cmp(balance) != 0 {
			log.Warn("balance does not match")
			continue
		}

		valid, err := checkProof(addr, slot, c)
		if err != nil {
			log.Warn(err)
		}
		if valid {
			log.Info("proof is valid!\n")
		} else {
			log.Warn("proof is invalid\n")
		}
		break
	}
}

func checkProof(contract common.Address, slot [32]byte, c *ethclient.Client) (bool, error) {
	keys := []string{fmt.Sprintf("%x", slot)}
	sproof, err := c.GetProof(context.TODO(), contract, keys, nil)
	//var cl rpc.Client
	//err := ec.c.CallContext(ctx, &result, "eth_getProof", account, keys, toBlockNumArg(blockNumber))
	if err != nil {
		return false, err
	}

	key, err := hex.DecodeString(strings.TrimPrefix(sproof.StorageProof[0].Key, "0x"))
	if err != nil {
		return false, err
	}
	log.Debugf("Key: %x", key)
	log.Debugf("Proof: %v", sproof.StorageProof[0].Proof)
	log.Debugf("RootHash: %s", sproof.StorageHash.String())

	var pvalue proof.RlpString
	pvalue, err = hex.DecodeString(strings.TrimPrefix(sproof.StorageProof[0].Value.String(), "0x"))
	if err != nil {
		return false, err
	}
	return proof.VerifyEthStorageProof(key, &pvalue, sproof.StorageHash.Bytes(), proof.ProofToBytes(sproof.StorageProof[0].Proof)), nil
}

func getSlot(holder string, position int) (slot [32]byte) {
	hl, err := hex.DecodeString(holder)
	if err != nil {
		log.Fatal(err)
	}
	hl = common.LeftPadBytes(hl, 32)
	posHex := fmt.Sprintf("%x", position)
	if len(posHex)%2 == 1 {
		posHex = "0" + posHex
	}
	p, err := hex.DecodeString(posHex)
	if err != nil {
		log.Fatal(err)
	}
	p = common.LeftPadBytes(p, 32)

	hash := crypto.Keccak256(hl, p)
	copy(slot[:], hash[:32])
	return
}

/*func verify(root common.Hash, key string, proof []string, kindex, pindex int, value []byte) (error, bool) {
	node, err := hex.DecodeString(proof[pindex])
	if err != nil {
		return err, false
	}
	dec := [][]byte{}

	if err = rlp.DecodeBytes(node, dec); err != nil {
		return err, false
	}
	if kindex == 0 {
		// trie root is always a hash
		if string(crypto.Keccak256(node)) != string(root.Bytes()) {
			return fmt.Errorf("root key do not match first hash"), false
		}
	} else if len(node) < 32 {
		if string(dec) != string(root.Bytes()) {
			return fmt.Errorf("node is a hash but it does not match with current root"), false
		}
	} else {
		if string(crypto.Keccak256(node)) != string(root.Bytes()) {
			return fmt.Errorf("root key do not match first hash"), false
		}
	}

	if len(dec) == 17 {
		// branch node
		if kindex >= len(key) {
			if string(dec[1:]) == string(value) {
				return nil, true
			}
		} else {
			i, err := strconv.ParseInt(string(key[kindex]), 16, 64) // not sure of this indexing
			if err != nil {
				return err, false
			}
			newRoot := dec[i]
			if string(newRoot) != string([]byte{}) {
				return verify(common.BytesToHash(newRoot), )
			}
		}
	}

	return nil, false
}
*/
