package rbhtrade

import (
	"errors"
	"math/big"
	"testing"
)

func TestMinOutputWithSlippage(t *testing.T) {
	expected := big.NewInt(1_001)
	minimum, err := MinOutputWithSlippage(expected, 500)
	if err != nil {
		t.Fatal(err)
	}
	if minimum.Cmp(big.NewInt(950)) != 0 {
		t.Fatalf("minimum = %s, want 950", minimum)
	}
	if expected.Cmp(big.NewInt(1_001)) != 0 {
		t.Fatal("expected output was mutated")
	}
	minimum, err = MinOutputWithSlippage(expected, BPSTotal)
	if err != nil || minimum.Sign() != 0 {
		t.Fatalf("full slippage minimum = %v, %v", minimum, err)
	}
	if _, err := MinOutputWithSlippage(nil, 1); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("nil output error = %v", err)
	}
	if _, err := MinOutputWithSlippage(expected, BPSTotal+1); !errors.Is(err, ErrInvalidBPS) {
		t.Fatalf("invalid slippage error = %v", err)
	}

	maximum := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	minimum, err = MinOutputWithSlippage(maximum, 0)
	if err != nil || minimum.Cmp(maximum) != 0 || minimum == maximum {
		t.Fatalf("maximum zero-slippage result = %v, %v", minimum, err)
	}
	tooLarge := new(big.Int).Add(maximum, big.NewInt(1))
	if _, err := MinOutputWithSlippage(tooLarge, 0); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("oversized output error = %v", err)
	}
}
