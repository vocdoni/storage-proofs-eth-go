package minime

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	"github.com/vocdoni/storage-proofs-eth-go/helpers"
)

// VerifyProof verifies a Minime storage proof.
// The targetBalance parameter is the full balance value, without decimals.
func VerifyProof(holder common.Address, storageRoot common.Hash,
	proofs []ethstorageproof.StorageResult, mapIndexSlot int, targetBalance,
	targetBlock *big.Int) error {
	// Sanity checks
	if len(proofs) != 2 {
		return fmt.Errorf("wrong length of storage proofs")
	}
	for _, p := range proofs {
		if p.Value == nil {
			return fmt.Errorf("value is nil")
		}
		if len(p.Key) != 32 {
			return fmt.Errorf("key length is wrong (%d)", len(p.Key))
		}
		if len(p.Proof) < 4 {
			return fmt.Errorf("proof length is wrong")
		}
	}
	if targetBalance == nil {
		return fmt.Errorf("target balance is nil")
	}
	if targetBlock == nil {
		return fmt.Errorf("target balance is nil")
	}

	// Check the proof keys (should match with the holder)
	if err := CheckMinimeKeys(proofs[0].Key, proofs[1].Key, holder, mapIndexSlot); err != nil {
		return fmt.Errorf("proof key and holder do not match: (%v)", err)
	}

	// Extract balance and block from the minime proof
	_, proof0Balance, proof0Block := ParseMinimeValue(proofs[0].Value, 1)
	if proof0Balance == nil || proof0Block == nil {
		return fmt.Errorf("cannot extract balance or block from the minime proof")
	}
	// Check balance matches with the provided balance
	if proof0Balance.Cmp(targetBalance) != 0 {
		return fmt.Errorf("proof balance and provided balance mismatch (%s != %s)",
			proof0Balance.String(), targetBalance.String())
	}

	// Proof 0 checkpoint block should be smaller or equal than target block
	if proof0Block.Cmp(targetBlock) > 1 { // p0 > t
		return fmt.Errorf("proof 0 checkpoint block is greather than the target block")
	}

	// Check if the proof1 is a proof of non existence (so proof0 is the last checkpoint).
	// If not the last, then check the target block is
	if len(proofs[1].Value) != 0 {
		_, _, proof1Block := ParseMinimeValue(proofs[1].Value, 1)
		if proof0Block.Cmp(proof1Block) >= 0 { // p0 >= p1
			return fmt.Errorf("proof 1 block is behind proof0 block")
		}
		if targetBlock.Cmp(proof1Block) >= 0 { // t >= p1
			return fmt.Errorf("proof 1 block number is smaller than target block")
		}
	}
	// Check both merkle proofs against the storage root hash
	for i, p := range proofs {
		valid, err := ethstorageproof.VerifyEthStorageProof(
			&ethstorageproof.StorageResult{
				Key:   p.Key,
				Proof: p.Proof,
				Value: p.Value,
			},
			storageRoot,
		)
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("proof %d is not valid", i)
		}
	}
	return nil
}

// ParseMinimeValue takes the value field from EIP1186 and splits into balance
// and block number (checkpoint). If decimals are unknown use 1.
//
// Returns the balance as big.Rat (considering the decimals), big.Int (not
// considering the decimals) and the Ethereum block number for the checkpoint.
func ParseMinimeValue(value []byte, decimals int) (*big.Rat, *big.Int, *big.Int) {
	// hexValue could be left zeroes trimed, so we need to expand it to 32 bytes
	value = common.LeftPadBytes(value, 32)
	mblock := new(big.Int).SetBytes(value[16:])
	ibalance := new(big.Int).SetBytes(value[:16])
	balance := helpers.BalanceToRat(ibalance, decimals)
	return balance, ibalance, mblock
}

// CheckMinimeKeys checks the validity of a storage proof key for a specific
// token holder address. As MiniMe includes checkpoints and each one adds +1 to
// the key, there is a maximum hardcoded tolerance of 2^16 positions for the
// key.
func CheckMinimeKeys(key1, key2 []byte, holder common.Address, mapIndexSlot int) error {
	mapSlot := helpers.GetMapSlot(holder, mapIndexSlot)
	vf := helpers.HashFromPosition(mapSlot)
	holderMapUindex := new(big.Int).SetBytes(vf[:])

	key1Uindex := new(big.Int).SetBytes(key1)
	key2Uindex := new(big.Int).SetBytes(key2)

	// key1+1 != key2
	if new(big.Int).Add(key1Uindex, big.NewInt(1)).Cmp(key2Uindex) != 0 {
		return fmt.Errorf("keys are not consecutive")
	}

	// We tolerate maximum 2^16 minime checkpoints
	offset := new(big.Int).Sub(key1Uindex, holderMapUindex)
	if offset.Cmp(big.NewInt(65536)) >= 0 || offset.Cmp(big.NewInt(0)) < 0 {
		return fmt.Errorf("key offset overflow")
	}
	return nil
}
