package mapbased

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/vocdoni/storage-proofs-eth-go/ethstorageproof"
	"github.com/vocdoni/storage-proofs-eth-go/helpers"
	"github.com/vocdoni/storage-proofs-eth-go/token/erc20"
)

const (
	DiscoveryIterations = 30
)

// ErrSlotNotFound represents the storage slot not found error
var ErrSlotNotFound = errors.New("storage slot not found")

// Mapbased tokens are those where the balance is stored on a map `address => uint256`.
// Most of ERC20 tokens follows this approach.
type Mapbased struct {
	erc20 *erc20.ERC20Token
}

// New creates a new Mapbased to get and verify Mapbased token proofs
func New(ctx context.Context, rpcCli *rpc.Client, tokenAddress common.Address) (*Mapbased, error) {
	erc20, err := erc20.New(ctx, rpcCli, tokenAddress)
	return &Mapbased{erc20: erc20}, err
}

// GetProof returns the storage merkle proofs for the acount holder.
func (m *Mapbased) GetProof(ctx context.Context, holder common.Address,
	block *big.Int, islot int) (*ethstorageproof.StorageProof, error) {
	return m.getMapProofWithIndexSlot(ctx, holder, block, islot)
}

// getMapProofWithIndexSlot returns the storage merkle proofs for the acount holder.
// The index slot is the position on the EVM storage sub-trie for the contract.
// If index slot is unknown, GetProof() could be used instead to try to find it
func (m *Mapbased) getMapProofWithIndexSlot(ctx context.Context, holder common.Address,
	block *big.Int, islot int) (*ethstorageproof.StorageProof, error) {
	slot := helpers.GetMapSlot(holder, islot)
	return m.erc20.GetProof(ctx, [][]byte{slot[:]}, block)
}

// DiscoverSlot tries to find the EVM storage index slot.
// A token holder address must be provided in order to have a balance to search and compare.
// Returns ErrSlotNotFound if the slot cannot be found.
// If found, returns also the amount stored.
func (m *Mapbased) DiscoverSlot(ctx context.Context, holder common.Address) (int, *big.Rat, error) {
	var slot [32]byte
	tokenData, err := m.erc20.GetTokenData(ctx)
	if err != nil {
		return -1, nil, fmt.Errorf("GetTokenData: %w", err)
	}
	balance, err := m.erc20.Balance(ctx, holder)
	if err != nil {
		return -1, nil, fmt.Errorf("balance: %w", err)
	}

	addr := common.Address{}
	copy(addr[:], m.erc20.TokenAddr[:20])

	var amount *big.Rat
	index := -1
	for i := 0; i < DiscoveryIterations; i++ {
		// Prepare storage index
		slot = helpers.GetMapSlot(holder, i)
		// Get Storage
		value, err := m.erc20.EthCli.StorageAt(ctx, addr, slot, nil)
		if err != nil {
			return index, nil, err
		}

		// Parse balance value
		amount, _ = helpers.ValueToBalance(value, int(tokenData.Decimals))
		// Check if balance matches
		if amount.Cmp(balance) == 0 {
			index = i
			break
		}
	}
	if index == -1 {
		return index, nil, ErrSlotNotFound
	}
	return index, amount, nil
}

// VerifyProof verifies a map based storage proof.
func (m *Mapbased) VerifyProof(holder common.Address, storageRoot common.Hash,
	proofs []ethstorageproof.StorageResult, mapIndexSlot int, targetBalance,
	targetBlock *big.Int) error {
	if len(proofs) != 1 {
		return fmt.Errorf("invalid length of proofs %d", len(proofs))
	}
	return VerifyProof(holder, storageRoot, proofs[0], mapIndexSlot, targetBalance, targetBlock)
}

// VerifyProof verifies a map based storage proof.
// The targetBalance parameter is the full balance value, without decimals.
func VerifyProof(holder common.Address, storageRoot common.Hash,
	proof ethstorageproof.StorageResult, mapIndexSlot int, targetBalance, targetBlock *big.Int) error {
	// Sanity checks
	if proof.Value == nil {
		return fmt.Errorf("value is nil")
	}
	if len(proof.Key) != 32 {
		return fmt.Errorf("key length is wrong.  Expected 32, got %v", len(proof.Key))
	}
	if targetBalance == nil {
		return fmt.Errorf("target balance is nil")
	}

	// Check proof key matches with holder address
	keySlot := helpers.GetMapSlot(holder, mapIndexSlot)
	if !bytes.Equal(keySlot[:], proof.Key) {
		return fmt.Errorf("proof key and leafData do not match (%x != %x)", keySlot, proof.Key)
	}

	// Check value balances matches
	proofBalance := new(big.Int).SetBytes(proof.Value)
	if targetBalance.Cmp(proofBalance) != 0 {
		return fmt.Errorf("proof balance and provided balance mismatch (%v != %v)",
			proofBalance, targetBalance)
	}

	// Check merkle proof against the storage root hash
	valid, err := ethstorageproof.VerifyEthStorageProof(
		&ethstorageproof.StorageResult{
			Key:   proof.Key,
			Proof: proof.Proof,
			Value: proof.Value,
		},
		storageRoot,
	)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("proof is not valid")
	}
	return nil
}
