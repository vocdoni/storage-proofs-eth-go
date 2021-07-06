package esps

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type ESPSstate struct {
	vars map[string]espType
}

type espType struct {
	typeVar reflect.Type
	value   interface{}
}

func (s *ESPSstate) setVar(name string, typ string, value interface{}) error {
	if name == "" || typ == "" {
		return fmt.Errorf("setVar: name or type are empty")
	}
	switch typ {
	case "uint32":
		v, ok := value.(uint32)
		if !ok {
			return fmt.Errorf("setVar: cannot convert argument into %s", typ)
		}
		s.vars[name] = espType{
			typeVar: reflect.TypeOf(v),
			value:   v,
		}
	case "uint64":
		v, ok := value.(uint64)
		if !ok {
			return fmt.Errorf("setVar: cannot convert argument into %s", typ)
		}
		s.vars[name] = espType{
			typeVar: reflect.TypeOf(v),
			value:   v,
		}
	case "uint256":
		fmt.Printf("TYPE: %s\n", reflect.TypeOf(value).String())
		v, ok := value.(*big.Int)
		if !ok {
			return fmt.Errorf("setVar: cannot convert argument into %s", typ)
		}
		s.vars[name] = espType{
			typeVar: reflect.TypeOf(v),
			value:   v,
		}
	case "uint128":
		v, ok := value.(*big.Int)
		if !ok {
			return fmt.Errorf("setVar: cannot convert argument into %s", typ)
		}
		s.vars[name] = espType{
			typeVar: reflect.TypeOf(v),
			value:   v,
		}
	case "address":
		v, ok := value.(common.Address)
		if !ok {
			return fmt.Errorf("setVar: cannot convert argument into %s", typ)
		}
		s.vars[name] = espType{
			typeVar: reflect.TypeOf(v),
			value:   v,
		}
	case "map":
		v, ok := value.(*espmap)
		if !ok {
			return fmt.Errorf("setVar: cannot convert argument into %s", typ)
		}
		s.vars[name] = espType{
			typeVar: reflect.TypeOf(v),
			value:   v,
		}
	}
	return nil
}

func (s *ESPSstate) getVar(name string) (*espType, error) {
	val, ok := s.vars[name]
	if !ok {
		return nil, fmt.Errorf("variable %s is unknown", name)
	}
	return &val, nil
}

func (s *ESPSstate) getValue(name string) (*espType, error) {
	namesplit := strings.Split(name, ".")
	val, err := s.getVar(namesplit[0])
	if err != nil {
		return nil, err
	}
	// If the name contains a dot, then its not a simple type (either and array or a map)
	if len(namesplit) > 1 {
		val2 := s.vars[namesplit[1]]
		switch val.typeVar.String() {

		case "*esps.espmap": // TODO: use a better way to identify vars
			key, err := VarToHex(&val2)
			// if a map, the second value is the map key
			if err != nil {
				return nil, err
			}
			keyAddress, err := val.value.(*espmap).GetValueAddress(key)
			if err != nil {
				return nil, err
			}
			storv := new(storageValue)
			storv.SetStorageAddress(keyAddress[:])

			return &espType{
				typeVar: reflect.TypeOf(storv),
				value:   storv,
			}, nil
		case "*esps.array": // TODO
		default:
			return nil, fmt.Errorf("not a complex type, cannot access to %s, type is %s", name, val.typeVar)
		}
	}
	return val, nil
}

func (s *ESPSstate) String() string {
	out := bytes.Buffer{}
	for name, value := range s.vars {
		out.WriteString(fmt.Sprintf("name:%s | type:%v | value:%v\n", name,
			value.typeVar.String(), reflect.ValueOf(value.value)))
	}
	return out.String()
}
