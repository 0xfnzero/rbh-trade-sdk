package pons

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestRobinhoodDeployment(t *testing.T) {
	rpcURL := os.Getenv("ROBINHOOD_RPC_URL")
	if rpcURL == "" {
		t.Skip("ROBINHOOD_RPC_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, eth, err := Dial(ctx, rpcURL)
	if err != nil {
		t.Fatal(err)
	}
	defer eth.Close()

	chainID, err := eth.ChainID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if chainID.Cmp(RobinhoodChainID) != 0 {
		t.Fatalf("chain id = %s, want %s", chainID, RobinhoodChainID)
	}

	addresses := client.Addresses()
	contracts := map[string]common.Address{
		"factory":             addresses.Factory,
		"meme hook":           addresses.MemeHook,
		"fee escrow":          addresses.FeeEscrow,
		"buyback vault":       addresses.BuybackVault,
		"launch locker":       addresses.LaunchLocker,
		"launch-and-buy":      addresses.LaunchAndBuy,
		"launch deployer":     addresses.LaunchDeployer,
		"graduation executor": addresses.GraduationExecutor,
		"graduation guard":    addresses.GraduationGuard,
	}
	for name, address := range contracts {
		code, err := eth.CodeAt(ctx, address, nil)
		if err != nil {
			t.Fatalf("read %s code: %v", name, err)
		}
		if len(code) == 0 {
			t.Errorf("%s has no deployed code at %s", name, address.Hex())
		}
	}

	if _, err := client.LaunchFee(ctx, nil); err != nil {
		t.Fatalf("read launch fee: %v", err)
	}
	if _, err := client.LaunchEnabled(ctx, nil); err != nil {
		t.Fatalf("read launch gate: %v", err)
	}
	if _, err := client.LaunchConfigCount(ctx, nil); err != nil {
		t.Fatalf("read launch config count: %v", err)
	}
}
