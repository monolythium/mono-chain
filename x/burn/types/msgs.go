package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewMsgBurn creates a new MsgBurn instance
func NewMsgBurn(fromAddress string, amount sdk.Coin) *MsgBurn {
	return &MsgBurn{
		FromAddress: fromAddress,
		Amount:      amount,
	}
}
