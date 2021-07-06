package esps

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEthProof(t *testing.T) {
	e := ESPS{}
	if err := e.Init(
		[]byte(jsonData2),
		common.HexToAddress("0xdac17f958d2ee523a2206206994597c13d831ec7"),
		"https://web3.dappnode.net"); err != nil {
		t.Fatal(err)
	}
	addr := common.HexToAddress("0xfbb1b73c4f0bda4f67dca266ce6ef42f520fbb98")
	if err := e.Run(uint32(2), addr); err != nil {
		t.Fatal(err)
	}
	t.Logf("\n%s", e.State.String())
}

var jsonData1 = string(`
{
	"name": "Minime",
	"logic": [
		{ "action": "setInput", "name": "index", "type": "uint32" },
		{ "action": "setInput", "name": "address", "type": "address" },
		{ "action": "setInput", "name": "checkpointPosition", "type": "uint32" },
		{ "action": "setMap", "name": "balanceMap", "slot": "index" },
		{ "action": "setArray", "name": "checkPoints", "slot": "balanceMap.address" },
		{ "action": "setUint256", "name": "block", "value": "checkPoints.checkpointPosition" },
		{ "action": "setUint128", "name": "height", "value": "block.0.128" },
		{ "action": "setUint128", "name": "balance", "value": "block.128.256" },
		{ "action": "setOutput", "name": "height" },
		{ "action": "setOutput", "name": "balance" }
	]
}
`)

var jsonData2 = string(`
{
	"name": "Mapbased",
	"logic": [
		{ "action": "setInput", "name": "index", "type": "uint32" },
		{ "action": "setInput", "name": "address", "type": "address" },
		{ "action": "setMap", "name": "balanceMap", "slot": "index" },
		{ "action": "setUint256", "name": "balance", "value": "balanceMap.address" },
		{ "action": "setOutput", "name": "balance" }
	]
}
`)
