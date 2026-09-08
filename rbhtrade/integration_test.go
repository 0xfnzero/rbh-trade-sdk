package rbhtrade

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

func TestRobinhoodDeployment(t *testing.T) {
	rpcURL := os.Getenv("ROBINHOOD_RPC_URL")
	if rpcURL == "" {
		t.Skip("set ROBINHOOD_RPC_URL to run the live deployment check")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	backend, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	if _, err := NewClientChecked(ctx, backend); err != nil {
		t.Fatal(err)
	}
}
