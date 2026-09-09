package rbhtrade

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
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

func TestRobinhoodBagsReads(t *testing.T) {
	rpcURL := os.Getenv("ROBINHOOD_RPC_URL")
	if rpcURL == "" {
		t.Skip("set ROBINHOOD_RPC_URL to run the live Bags read check")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	backend, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

	token := common.HexToAddress("0xdb89f57b36e2778d1c7a42f8cda4a5bf496e76ae")
	stateCall, err := BuildBagsGetTokenState(token)
	if err != nil {
		t.Fatal(err)
	}
	stateData, err := backend.CallContract(ctx, ethereum.CallMsg{To: &stateCall.To, Data: stateCall.Data}, nil)
	if err != nil {
		t.Fatal(err)
	}
	state, err := DecodeBagsTokenState(stateData)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Exists || state.Curve == (common.Address{}) || state.PoolID == (common.Hash{}) {
		t.Fatalf("incomplete Bags token state: %#v", state)
	}

	if !state.Migrated {
		quoteCall, err := BuildBagsQuoteBuy(state.Curve, big.NewInt(1_000_000))
		if err != nil {
			t.Fatal(err)
		}
		quoteData, err := backend.CallContract(ctx, ethereum.CallMsg{To: &quoteCall.To, Data: quoteCall.Data}, nil)
		if err != nil {
			t.Fatal(err)
		}
		quote, err := DecodeBagsQuoteBuy(quoteData)
		if err != nil {
			t.Fatal(err)
		}
		if quote.TokensOut.Sign() <= 0 || new(big.Int).Add(quote.GrossUsed, quote.RefundQuote).Cmp(big.NewInt(1_000_000)) != 0 {
			t.Fatalf("invalid Bags buy quote: %#v", quote)
		}
	}

	claimCall, err := BuildBagsClaimableOf(token, token)
	if err != nil {
		t.Fatal(err)
	}
	claimData, err := backend.CallContract(ctx, ethereum.CallMsg{To: &claimCall.To, Data: claimCall.Data}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeBagsClaimableOf(claimData); err != nil {
		t.Fatal(err)
	}
}
