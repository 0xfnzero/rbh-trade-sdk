package rbhtrade

import (
	"context"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type deploymentBackendStub struct {
	mu      sync.Mutex
	checked map[common.Address]struct{}
	missing common.Address
}

func (b *deploymentBackendStub) ChainID(context.Context) (*big.Int, error) {
	return big.NewInt(ChainID), nil
}

func (b *deploymentBackendStub) CodeAt(_ context.Context, address common.Address, _ *big.Int) ([]byte, error) {
	b.mu.Lock()
	b.checked[address] = struct{}{}
	b.mu.Unlock()
	if address == b.missing {
		return nil, nil
	}
	return []byte{1}, nil
}

func TestNewClientCheckedCoversBuilderTargets(t *testing.T) {
	backend := &deploymentBackendStub{checked: make(map[common.Address]struct{})}
	if _, err := NewClientChecked(context.Background(), backend); err != nil {
		t.Fatal(err)
	}
	want := DefaultAddressBook()
	for name, address := range map[string]common.Address{
		"PoolManager":       want.PoolManager,
		"PositionManager":   want.PositionManager,
		"V4Quoter":          want.V4Quoter,
		"StateView":         want.StateView,
		"UniversalRouter":   want.UniversalRouter,
		"Permit2":           want.Permit2,
		"WETH":              want.WETH,
		"USDG":              want.USDG,
		"V3Factory":         want.V3Factory,
		"V3SwapRouter02":    want.V3SwapRouter02,
		"PonsFactory":       want.PonsFactory,
		"PonsLaunchAndBuy":  want.PonsLaunchAndBuy,
		"PonsMemeHook":      want.PonsMemeHook,
		"LongLauncher":      want.LongLauncher,
		"LongAirlock":       want.LongAirlock,
		"LongTokenFactory":  want.LongTokenFactory,
		"LongDopplerHook":   want.LongDopplerHook,
		"O1Factory":         want.O1Factory,
		"O1Hook":            want.O1Hook,
		"O1FeeEscrow":       want.O1FeeEscrow,
		"O1TokenDeployer":   want.O1TokenDeployer,
		"O1Registry":        want.O1Registry,
		"O1LaunchBuy":       want.O1LaunchBuy,
		"PoolsEntry":        want.PoolsEntry,
		"PoolsTokenFactory": want.PoolsTokenFactory,
		"PoolsInstant":      want.PoolsInstant,
		"PAIRLaunchpad":     want.PAIRLaunchpad,
		"PAIRLocker":        want.PAIRLocker,
		"BagsFactory":       want.BagsFactory,
		"BagsHook":          want.BagsHook,
		"BagsVault":         want.BagsVault,
		"BagsLens":          want.BagsLens,
	} {
		if _, ok := backend.checked[address]; !ok {
			t.Errorf("%s was not checked", name)
		}
	}
}

func TestNewClientCheckedRejectsMissingBuilderTarget(t *testing.T) {
	backend := &deploymentBackendStub{
		checked: make(map[common.Address]struct{}),
		missing: DefaultAddressBook().V4Quoter,
	}
	if _, err := NewClientChecked(context.Background(), backend); err == nil {
		t.Fatal("missing V4Quoter code was accepted")
	}
}

func TestNewClientRejectsNilInputs(t *testing.T) {
	var backend *deploymentBackendStub
	if _, err := NewClient(context.Background(), backend); err == nil {
		t.Fatal("typed-nil backend was accepted")
	}
	backend = &deploymentBackendStub{}
	//lint:ignore SA1012 This call verifies the public constructor's nil guard.
	if _, err := NewClient(nil, backend); err == nil {
		t.Fatal("nil context was accepted")
	}
}
