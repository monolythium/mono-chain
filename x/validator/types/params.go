package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewParams(
	validatorRegistrationBurn sdk.Coin,
	validatorMinSelfDelegation sdk.Coin,
) Params {
	return Params{
		ValidatorRegistrationBurn:  validatorRegistrationBurn,
		ValidatorMinSelfDelegation: validatorMinSelfDelegation,
	}
}

func DefaultParams() Params {
	return NewParams(
		sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
		sdk.NewCoin(sdk.DefaultBondDenom, math.ZeroInt()),
	)
}

func (p Params) Validate() error {
	if err := validateRegistrationBurn(p.ValidatorRegistrationBurn); err != nil {
		return err
	}

	return validateMinSelfDelegation(p.ValidatorMinSelfDelegation)
}

func validateRegistrationBurn(v sdk.Coin) error {
	if err := v.Validate(); err != nil {
		return ErrInvalidRegistrationBurn
	}

	if !v.IsZero() && v.Denom != sdk.DefaultBondDenom {
		return ErrInvalidRegistrationBurn
	}

	return nil
}

func validateMinSelfDelegation(v sdk.Coin) error {
	if err := v.Validate(); err != nil {
		return ErrInvalidMinSelfDelegation
	}

	if !v.IsZero() && v.Denom != sdk.DefaultBondDenom {
		return ErrInvalidMinSelfDelegation
	}

	return nil
}
