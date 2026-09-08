package pons

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type offlineTransactor struct{}

func (offlineTransactor) EstimateGas(context.Context, ethereum.CallMsg) (uint64, error) {
	return 0, errors.New("unexpected EstimateGas call")
}

func (offlineTransactor) SuggestGasPrice(context.Context) (*big.Int, error) {
	return nil, errors.New("unexpected SuggestGasPrice call")
}

func (offlineTransactor) SuggestGasTipCap(context.Context) (*big.Int, error) {
	return nil, errors.New("unexpected SuggestGasTipCap call")
}

func (offlineTransactor) SendTransaction(context.Context, *types.Transaction) error {
	return errors.New("unexpected SendTransaction call")
}

func (offlineTransactor) HeaderByNumber(context.Context, *big.Int) (*types.Header, error) {
	return nil, errors.New("unexpected HeaderByNumber call")
}

func (offlineTransactor) PendingCodeAt(context.Context, common.Address) ([]byte, error) {
	return nil, errors.New("unexpected PendingCodeAt call")
}

func (offlineTransactor) PendingNonceAt(context.Context, common.Address) (uint64, error) {
	return 0, errors.New("unexpected PendingNonceAt call")
}

func (offlineTransactor) TransactionByHash(context.Context, common.Hash) (*types.Transaction, bool, error) {
	return nil, false, errors.New("unexpected TransactionByHash call")
}

func TestClientTransactionWrappers(t *testing.T) {
	addresses := Addresses{
		Factory:      common.HexToAddress("0x00000000000000000000000000000000000000f1"),
		MemeHook:     common.HexToAddress("0x00000000000000000000000000000000000000f2"),
		FeeEscrow:    common.HexToAddress("0x00000000000000000000000000000000000000f3"),
		BuybackVault: common.HexToAddress("0x00000000000000000000000000000000000000f4"),
		LaunchAndBuy: common.HexToAddress("0x00000000000000000000000000000000000000f5"),
	}
	client := &Client{transactor: offlineTransactor{}, addresses: addresses}
	curve := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	token := common.HexToAddress("0x00000000000000000000000000000000000000c2")
	spender := common.HexToAddress("0x00000000000000000000000000000000000000c3")
	recipient := common.HexToAddress("0x00000000000000000000000000000000000000c4")
	deployer := common.HexToAddress("0x00000000000000000000000000000000000000c5")
	pairToken := common.HexToAddress("0x00000000000000000000000000000000000000c6")
	poolID := common.HexToHash("0x1234")
	launchConfigID := big.NewInt(3)
	quoteIn := big.NewInt(13)
	minOut := big.NewInt(7)
	amount := big.NewInt(19)
	exemptions := []common.Address{recipient, deployer}
	params := TokenParams{
		Name:                "Example",
		Symbol:              "EXMPL",
		CreatorFeeRecipient: recipient,
		CreatorTaxBps:       100,
		BuybackEnabled:      true,
	}
	launchFee := big.NewInt(11)
	launchAndBuyValue := new(big.Int).Add(new(big.Int).Set(launchFee), quoteIn)

	type transactionCall func(*bind.TransactOpts) (*types.Transaction, error)
	type packCall func() ([]byte, error)
	tests := []struct {
		name  string
		to    common.Address
		value *big.Int
		call  transactionCall
		pack  packCall
	}{
		{"LaunchTokenSimple", addresses.Factory, launchFee, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.LaunchTokenSimple(auth, params, launchConfigID, pairToken)
		}, func() ([]byte, error) { return PackLaunchTokenSimple(params, launchConfigID, pairToken) }},
		{"LaunchToken", addresses.Factory, launchFee, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.LaunchToken(auth, params, launchConfigID, pairToken, exemptions)
		}, func() ([]byte, error) { return PackLaunchToken(params, launchConfigID, pairToken, exemptions) }},
		{"LaunchTokenFor", addresses.Factory, launchFee, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.LaunchTokenFor(auth, params, launchConfigID, pairToken, deployer, exemptions)
		}, func() ([]byte, error) {
			return PackLaunchTokenFor(params, launchConfigID, pairToken, deployer, exemptions)
		}},
		{"LaunchAndBuy", addresses.LaunchAndBuy, launchAndBuyValue, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.LaunchAndBuy(auth, params, launchConfigID, pairToken, quoteIn, minOut, recipient, exemptions)
		}, func() ([]byte, error) {
			return PackLaunchAndBuy(params, launchConfigID, pairToken, quoteIn, minOut, recipient, exemptions)
		}},
		{"Buy", curve, quoteIn, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.Buy(auth, curve, quoteIn, minOut, recipient)
		}, func() ([]byte, error) { return PackBuy(quoteIn, minOut, recipient) }},
		{"Sell", curve, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.Sell(auth, curve, amount, minOut, recipient)
		}, func() ([]byte, error) { return PackSell(amount, minOut, recipient) }},
		{"SweepCurveFees", curve, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.SweepCurveFees(auth, curve, minOut)
		}, func() ([]byte, error) { return PackSweepCurveFees(minOut) }},
		{"Graduate", addresses.Factory, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.Graduate(auth, token)
		}, func() ([]byte, error) { return PackGraduate(token) }},
		{"CreateGraduatedPool", addresses.Factory, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.CreateGraduatedPool(auth, token)
		}, func() ([]byte, error) { return PackCreateGraduatedPool(token) }},
		{"SweepPoolFees", addresses.MemeHook, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.SweepPoolFees(auth, poolID, amount, minOut)
		}, func() ([]byte, error) { return PackSweepPoolFees(poolID, amount, minOut) }},
		{"TransferCreatorFeeRecipient", addresses.Factory, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.TransferCreatorFeeRecipient(auth, token, recipient)
		}, func() ([]byte, error) { return PackTransferCreatorFeeRecipient(token, recipient) }},
		{"SetBuybackEnabled", addresses.Factory, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.SetBuybackEnabled(auth, token, true)
		}, func() ([]byte, error) { return PackSetBuybackEnabled(token, true) }},
		{"Approve", token, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.Approve(auth, token, spender, amount)
		}, func() ([]byte, error) { return PackApprove(spender, amount) }},
		{"ClaimNative", addresses.FeeEscrow, nil, client.ClaimNative, PackClaimNative},
		{"ClaimNativeAmount", addresses.FeeEscrow, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.ClaimNativeAmount(auth, amount)
		}, func() ([]byte, error) { return PackClaimNativeAmount(amount) }},
		{"ClaimToken", addresses.FeeEscrow, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.ClaimToken(auth, token)
		}, func() ([]byte, error) { return PackClaimToken(token) }},
		{"ClaimTokenAmount", addresses.FeeEscrow, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.ClaimTokenAmount(auth, token, amount)
		}, func() ([]byte, error) { return PackClaimTokenAmount(token, amount) }},
		{"ReleaseVested", addresses.BuybackVault, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.ReleaseVested(auth, token)
		}, func() ([]byte, error) { return PackReleaseVested(token) }},
		{"ReleaseVest", addresses.BuybackVault, nil, func(auth *bind.TransactOpts) (*types.Transaction, error) {
			return client.ReleaseVest(auth, token)
		}, func() ([]byte, error) { return PackReleaseVested(token) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			auth := &bind.TransactOpts{
				From:     deployer,
				Nonce:    big.NewInt(7),
				Signer:   func(_ common.Address, tx *types.Transaction) (*types.Transaction, error) { return tx, nil },
				Value:    tc.value,
				GasPrice: big.NewInt(1),
				GasLimit: 500_000,
				NoSend:   true,
			}
			tx, err := tc.call(auth)
			if err != nil {
				t.Fatal(err)
			}
			wantData, err := tc.pack()
			if err != nil {
				t.Fatal(err)
			}
			if tx.To() == nil || *tx.To() != tc.to {
				t.Fatalf("transaction recipient = %v, want %s", tx.To(), tc.to)
			}
			if !bytes.Equal(tx.Data(), wantData) {
				t.Fatalf("transaction calldata = 0x%x, want 0x%x", tx.Data(), wantData)
			}
			wantValue := new(big.Int)
			if tc.value != nil {
				wantValue.Set(tc.value)
			}
			if tx.Value().Cmp(wantValue) != 0 {
				t.Fatalf("transaction value = %s, want %s", tx.Value(), wantValue)
			}
		})
	}
}

func TestNonPayableTransactionsRejectValue(t *testing.T) {
	client := &Client{transactor: offlineTransactor{}}
	auth := &bind.TransactOpts{Value: big.NewInt(1)}
	_, err := client.Sell(auth, common.HexToAddress("0x1"), big.NewInt(1), big.NewInt(0), common.HexToAddress("0x2"))
	if !errors.Is(err, ErrUnexpectedTransactionValue) {
		t.Fatalf("expected ErrUnexpectedTransactionValue, got %v", err)
	}
}

func TestNilClientTransactionReturnsError(t *testing.T) {
	var client *Client
	auth := &bind.TransactOpts{}
	_, err := client.LaunchTokenSimple(auth, TokenParams{Name: "N", Symbol: "S"}, big.NewInt(0), NativeQuote)
	if !errors.Is(err, ErrNilBackend) {
		t.Fatalf("expected ErrNilBackend, got %v", err)
	}
	if got := client.Addresses(); got != (Addresses{}) {
		t.Fatalf("nil client addresses = %#v", got)
	}
}
