package rbhtrade

import "errors"

var (
	ErrInvalidAddress     = errors.New("invalid zero address")
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrInvalidPoolKey     = errors.New("invalid pool key")
	ErrCurrencyNotInPool  = errors.New("currency is not in pool")
	ErrSameCurrency       = errors.New("input and output currencies are equal")
	ErrExpiredDeadline    = errors.New("deadline is zero")
	ErrInvalidChainID     = errors.New("unexpected chain id")
	ErrMismatchedLengths  = errors.New("slice lengths do not match")
	ErrInvalidBPS         = errors.New("invalid basis points")
	ErrInvalidBagsQuote   = errors.New("invalid Bags quote")
	ErrInvalidBagsState   = errors.New("invalid Bags token state")
	ErrInvalidTransaction = errors.New("invalid transaction parameters")
	ErrNoBroadcasters     = errors.New("no transaction broadcasters")
	ErrBroadcastFailed    = errors.New("transaction broadcast failed")
	ErrBroadcastRejected  = errors.New("transaction was rejected by every broadcaster")
	ErrBroadcastUnknown   = errors.New("transaction broadcast outcome is unknown")
)
