package esps

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/vocdoni/storage-proofs-eth-go/token/erc20"
)

type ESPS struct {
	Name  string     `json:"name"`
	Logic []Action   `json:"logic"`
	State *ESPSstate `json:"-"`

	erc20 *erc20.ERC20Token `json:"-"`
}

type Action struct {
	Action string `json:"action"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Slot   string `json:"slot"`
	Value  string `json:"value"`
}

/*
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
*/

func (e *ESPS) Init(jsonData []byte, contractAddr common.Address, web3 string) error {
	if err := json.Unmarshal(jsonData, e); err != nil {
		return err
	}
	if web3 != "" {
		e.erc20 = &erc20.ERC20Token{}
		if err := e.erc20.Init(context.Background(), web3, contractAddr.Hex()); err != nil {
			return err
		}
	}
	return nil
}

func (e *ESPS) Run(args ...interface{}) error {
	e.State = new(ESPSstate)
	e.State.vars = make(map[string]espType)
	argsIndex := 0
	for _, act := range e.Logic {
		switch act.Action {
		case "setInput":
			if err := e.State.setVar(act.Name, act.Type, args[argsIndex]); err != nil {
				return err
			}
			argsIndex++
		case "setMap":
			value, err := e.State.getVar(act.Slot)
			if err != nil {
				return err
			}
			hexSlot, err := VarToHex(value)
			if err != nil {
				return err
			}
			fmt.Printf("hexSlot is %s\n", hexSlot)
			m := new(espmap)
			if err := m.setSlot(hexSlot); err != nil {
				return err
			}
			if err := e.State.setVar(act.Name, "map", m); err != nil {
				return err
			}
		case "setArray":
		case "setUint256":
			value, err := e.State.getValue(act.Value)
			if err != nil {
				return err
			}
			v := value.value
			if value.typeVar.String() == "*esps.storageValue" {
				var sv []byte
				if sv, err = value.value.(*storageValue).Get(e.erc20); err != nil {
					return err
				}
				v = new(big.Int).SetBytes(sv)
			}
			if err := e.State.setVar(act.Name, "uint256", v); err != nil {
				return err
			}
		case "setUint128":
		case "setOutput":
			value, err := e.State.getValue(act.Name)
			if err != nil {
				return err
			}
			fmt.Printf("OUTPUT %s: %v", act.Name, value.value)
		default:
			return fmt.Errorf("action %s is unknown", act.Action)
		}
	}
	return nil
}
