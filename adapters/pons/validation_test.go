package pons

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestCalldataValidationRejectsUnsafeInputs(t *testing.T) {
	address := common.HexToAddress("0x0000000000000000000000000000000000000001")
	overflow := new(big.Int).Lsh(big.NewInt(1), 256)
	validParams := TokenParams{Name: "Example", Symbol: "EX"}
	tooManyExemptions := make([]common.Address, MaxSnipeTaxExemptions+1)
	tests := []struct {
		name string
		call func() ([]byte, error)
	}{
		{"empty token name", func() ([]byte, error) {
			return PackLaunchToken(TokenParams{Symbol: "EX"}, big.NewInt(0), NativeQuote, nil)
		}},
		{"too many exemptions", func() ([]byte, error) {
			return PackLaunchToken(validParams, big.NewInt(0), NativeQuote, tooManyExemptions)
		}},
		{"nil launch config", func() ([]byte, error) { return PackLaunchToken(validParams, nil, NativeQuote, nil) }},
		{"negative launch config", func() ([]byte, error) { return PackLaunchTokenSimple(validParams, big.NewInt(-1), NativeQuote) }},
		{"overflow launch config", func() ([]byte, error) { return PackLaunchTokenSimple(validParams, overflow, NativeQuote) }},
		{"nil buy input", func() ([]byte, error) { return PackBuy(nil, big.NewInt(0), address) }},
		{"zero buy input", func() ([]byte, error) { return PackBuy(big.NewInt(0), big.NewInt(0), address) }},
		{"negative min output", func() ([]byte, error) { return PackSell(big.NewInt(1), big.NewInt(-1), address) }},
		{"zero recipient", func() ([]byte, error) { return PackBuy(big.NewInt(1), big.NewInt(0), common.Address{}) }},
		{"zero token", func() ([]byte, error) { return PackGraduate(common.Address{}) }},
		{"zero spender", func() ([]byte, error) { return PackApprove(common.Address{}, big.NewInt(1)) }},
		{"nil claim amount", func() ([]byte, error) { return PackClaimNativeAmount(nil) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.call(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestTransactionValidationRejectsNilBackendAuthAndDestinations(t *testing.T) {
	address := common.HexToAddress("0x0000000000000000000000000000000000000001")
	auth := &bind.TransactOpts{
		From:     address,
		Nonce:    big.NewInt(0),
		Signer:   func(_ common.Address, tx *types.Transaction) (*types.Transaction, error) { return tx, nil },
		Value:    big.NewInt(10),
		GasPrice: big.NewInt(1),
		GasLimit: 100_000,
		NoSend:   true,
	}

	if _, err := NewClient(nil).Buy(auth, address, big.NewInt(1), big.NewInt(0), address); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("nil backend error = %v", err)
	}
	client := &Client{transactor: offlineTransactor{}, addresses: Addresses{}}
	if _, err := client.Buy(nil, address, big.NewInt(1), big.NewInt(0), address); !errors.Is(err, ErrNilAuth) {
		t.Fatalf("nil auth error = %v", err)
	}
	if _, err := client.Buy(auth, common.Address{}, big.NewInt(1), big.NewInt(0), address); !errors.Is(err, ErrZeroAddress) {
		t.Fatalf("zero curve error = %v", err)
	}
	if _, err := client.LaunchTokenSimple(auth, TokenParams{}, big.NewInt(0), NativeQuote); !errors.Is(err, ErrZeroAddress) {
		t.Fatalf("zero factory error = %v", err)
	}
	if _, err := client.LaunchAndBuy(auth, TokenParams{}, big.NewInt(0), NativeQuote, big.NewInt(1), big.NewInt(0), address, nil); !errors.Is(err, ErrZeroAddress) {
		t.Fatalf("zero router error = %v", err)
	}
}

func TestReadValidationRejectsNilBackendAndZeroTargets(t *testing.T) {
	ctx := context.Background()
	if _, err := NewClient(nil).LaunchFee(ctx, nil); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("nil backend error = %v", err)
	}
	client := &Client{caller: &readCaller{responses: make(map[readResponseKey][]byte)}}
	if _, err := client.LaunchFee(ctx, nil); !errors.Is(err, ErrZeroAddress) {
		t.Fatalf("zero factory error = %v", err)
	}
	if _, err := client.GetReserves(ctx, common.Address{}, nil); !errors.Is(err, ErrZeroAddress) {
		t.Fatalf("zero curve error = %v", err)
	}
	if _, err := client.TokenInfo(ctx, common.Address{}, nil); !errors.Is(err, ErrZeroAddress) {
		t.Fatalf("zero token error = %v", err)
	}
	if _, err := client.GetLaunchConfig(ctx, nil, nil); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("nil launch config error = %v", err)
	}
}

func TestValidateChainID(t *testing.T) {
	if err := validateChainID(big.NewInt(ChainID)); err != nil {
		t.Fatal(err)
	}
	for _, chainID := range []*big.Int{nil, big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 80)} {
		if err := validateChainID(chainID); !errors.Is(err, ErrWrongChain) {
			t.Fatalf("chain ID %v: expected ErrWrongChain, got %v", chainID, err)
		}
	}
}
