package keeper_test

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"go.uber.org/mock/gomock"

	"github.com/monolythium/mono-chain/x/burn/keeper"
	"github.com/monolythium/mono-chain/x/burn/types"
)

func (s *BurnKeeperTestSuite) TestMsgBurn_HappyPath() {
	addr := sdk.AccAddress("test_msg_burn_ok____")
	addrStr, err := s.addressCodec.BytesToString(addr)
	s.Require().NoError(err)

	gomock.InOrder(s.expectBurnCalls(addr, 100)...)

	resp, err := s.msgServer.Burn(s.ctx, &types.MsgBurn{
		FromAddress: addrStr,
		Amount:      sdk.NewCoin("alyth", math.NewInt(100)),
	})
	s.Require().NoError(err)
	s.Require().NotNil(resp)
}

func (s *BurnKeeperTestSuite) TestMsgBurn_InvalidAddress() {
	_, err := s.msgServer.Burn(s.ctx, &types.MsgBurn{
		FromAddress: "not_valid_bech32",
		Amount:      sdk.NewCoin("alyth", math.NewInt(100)),
	})
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestMsgBurn_KeeperError() {
	addr := sdk.AccAddress("test_msg_burn_err___")
	addrStr, err := s.addressCodec.BytesToString(addr)
	s.Require().NoError(err)

	// BurnFromAccount transfers first, then burnNative rejects the denom
	coins := sdk.NewCoins(sdk.NewCoin("uatom", math.NewInt(100)))
	s.bankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), addr, "burn", coins).Return(nil)

	_, err = s.msgServer.Burn(s.ctx, &types.MsgBurn{
		FromAddress: addrStr,
		Amount:      sdk.NewCoin("uatom", math.NewInt(100)),
	})
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrInvalidBurnDenom)
}

func (s *BurnKeeperTestSuite) TestMsgUpdateParams_HappyPath() {
	newParams := types.NewParams(math.LegacyNewDecWithPrec(3, 1)) // 0.3

	resp, err := s.msgServer.UpdateParams(s.ctx, &types.MsgUpdateParams{
		Authority: s.authority,
		Params:    newParams,
	})
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	got, err := s.burnKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(newParams, got)
}

func (s *BurnKeeperTestSuite) TestMsgUpdateParams_WrongAuthority() {
	_, err := s.msgServer.UpdateParams(s.ctx, &types.MsgUpdateParams{
		Authority: "mono1wrongauthority",
		Params:    types.DefaultParams(),
	})
	s.Require().Error(err)
	s.Require().ErrorIs(err, govtypes.ErrInvalidSigner)
}

func (s *BurnKeeperTestSuite) TestMsgUpdateParams_ParamsSetFails() {
	sb := collections.NewSchemaBuilder(failSetStoreService{})
	s.burnKeeper.Params = collections.NewItem(sb, types.ParamsKey, "params",
		codec.CollValue[types.Params](s.encCfg.Codec))
	_, _ = sb.Build()
	// Recreate msgServer with the modified keeper
	s.msgServer = keeper.NewMsgServerImpl(s.burnKeeper)

	_, err := s.msgServer.UpdateParams(s.ctx, &types.MsgUpdateParams{
		Authority: s.authority,
		Params:    types.DefaultParams(),
	})
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestMsgUpdateParams_InvalidParams() {
	// Negative fee burn percent
	_, err := s.msgServer.UpdateParams(s.ctx, &types.MsgUpdateParams{
		Authority: s.authority,
		Params:    types.NewParams(math.LegacyNewDec(-1)),
	})
	s.Require().Error(err)
	s.Require().ErrorIs(err, types.ErrInvalidFeeBurnPercent)
}
