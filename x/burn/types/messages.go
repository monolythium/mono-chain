package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgBurn{}

// NewMsgBurn creates a new MsgBurn instance
func NewMsgBurn(fromAddress string, amount sdk.Coin) *MsgBurn {
	return &MsgBurn{
		FromAddress: fromAddress,
		Amount:      amount,
	}
}

// ValidateBasic performs stateless validation
func (msg *MsgBurn) ValidateBasic() error {
	// Validate address
	_, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid from address: %s", err)
	}

	// Validate amount
	if !msg.Amount.IsValid() || !msg.Amount.IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "invalid amount")
	}

	// SECURITY: Only allow burning of native token
	if msg.Amount.Denom != sdk.DefaultBondDenom {
		return errors.Wrapf(ErrInvalidBurnDenom, "can only burn %s, got %s", sdk.DefaultBondDenom, msg.Amount.Denom)
	}

	return nil
}

// GetSigners returns the expected signers for MsgBurn
func (msg *MsgBurn) GetSigners() []sdk.AccAddress {
	sender, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{sender}
}
