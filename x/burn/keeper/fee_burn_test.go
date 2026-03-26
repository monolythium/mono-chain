package keeper_test

import (
	"errors"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/mock/gomock"

	"github.com/monolythium/mono-chain/x/burn/types"
)

func (s *BurnKeeperTestSuite) TestProcessFeeBurn_ParamsNotSetReturnsError() {
	// Remove params so Params.Get returns ErrNotFound
	err := s.burnKeeper.Params.Remove(s.ctx)
	s.Require().NoError(err)

	err = s.burnKeeper.ProcessFeeBurn(s.ctx)
	s.Require().Error(err)
}

func (s *BurnKeeperTestSuite) TestProcessFeeBurn_HappyPath50Percent() {
	params := types.NewParams(math.LegacyNewDecWithPrec(5, 1)) // 0.5
	err := s.burnKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)

	fcAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	burnCoins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(50)))

	gomock.InOrder(
		s.bankKeeper.EXPECT().GetBalance(gomock.Any(), fcAddr, "alyth").Return(sdk.NewCoin("alyth", math.NewInt(100))),
		s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), authtypes.FeeCollectorName, "burn", burnCoins).Return(nil),
		s.bankKeeper.EXPECT().BurnCoins(gomock.Any(), "burn", burnCoins).Return(nil),
	)

	err = s.burnKeeper.ProcessFeeBurn(s.ctx)
	s.Require().NoError(err)

	count, err := s.burnKeeper.GlobalBurnCount.Peek(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1), count)

	total, err := s.burnKeeper.GlobalBurnTotal.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(50), total)
}

func (s *BurnKeeperTestSuite) TestProcessFeeBurn_ZeroFees() {
	params := types.NewParams(math.LegacyNewDecWithPrec(5, 1))
	err := s.burnKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)

	fcAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	s.bankKeeper.EXPECT().GetBalance(gomock.Any(), fcAddr, "alyth").Return(sdk.NewCoin("alyth", math.ZeroInt()))

	err = s.burnKeeper.ProcessFeeBurn(s.ctx)
	s.Require().NoError(err)
}

func (s *BurnKeeperTestSuite) TestProcessFeeBurn_ZeroPercent() {
	// Default params have FeeBurnPercent = 0
	fcAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	s.bankKeeper.EXPECT().GetBalance(gomock.Any(), fcAddr, "alyth").Return(sdk.NewCoin("alyth", math.NewInt(100)))

	err := s.burnKeeper.ProcessFeeBurn(s.ctx)
	s.Require().NoError(err)
}

func (s *BurnKeeperTestSuite) TestProcessFeeBurn_100Percent() {
	params := types.NewParams(math.LegacyOneDec())
	err := s.burnKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)

	fcAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	burnCoins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(100)))

	gomock.InOrder(
		s.bankKeeper.EXPECT().GetBalance(gomock.Any(), fcAddr, "alyth").Return(sdk.NewCoin("alyth", math.NewInt(100))),
		s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), authtypes.FeeCollectorName, "burn", burnCoins).Return(nil),
		s.bankKeeper.EXPECT().BurnCoins(gomock.Any(), "burn", burnCoins).Return(nil),
	)

	err = s.burnKeeper.ProcessFeeBurn(s.ctx)
	s.Require().NoError(err)

	total, err := s.burnKeeper.GlobalBurnTotal.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(100), total)
}

func (s *BurnKeeperTestSuite) TestProcessFeeBurn_FractionalTruncation() {
	params := types.NewParams(math.LegacyNewDecWithPrec(5, 1)) // 0.5
	err := s.burnKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)

	fcAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	// 0.5 * 33 = 16.5, truncated to 16
	burnCoins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(16)))

	gomock.InOrder(
		s.bankKeeper.EXPECT().GetBalance(gomock.Any(), fcAddr, "alyth").Return(sdk.NewCoin("alyth", math.NewInt(33))),
		s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), authtypes.FeeCollectorName, "burn", burnCoins).Return(nil),
		s.bankKeeper.EXPECT().BurnCoins(gomock.Any(), "burn", burnCoins).Return(nil),
	)

	err = s.burnKeeper.ProcessFeeBurn(s.ctx)
	s.Require().NoError(err)

	total, err := s.burnKeeper.GlobalBurnTotal.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(math.NewInt(16), total)
}

func (s *BurnKeeperTestSuite) TestProcessFeeBurn_SendModuleToModuleFails() {
	params := types.NewParams(math.LegacyNewDecWithPrec(5, 1))
	err := s.burnKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)

	fcAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	sendErr := errors.New("send module to module failed")
	burnCoins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(50)))

	gomock.InOrder(
		s.bankKeeper.EXPECT().GetBalance(gomock.Any(), fcAddr, "alyth").Return(sdk.NewCoin("alyth", math.NewInt(100))),
		s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), authtypes.FeeCollectorName, "burn", burnCoins).Return(sendErr),
	)

	err = s.burnKeeper.ProcessFeeBurn(s.ctx)
	s.Require().Error(err)
	s.Require().ErrorIs(err, sendErr)
}

func (s *BurnKeeperTestSuite) TestProcessFeeBurn_BurnCoinsFails() {
	params := types.NewParams(math.LegacyNewDecWithPrec(5, 1))
	err := s.burnKeeper.Params.Set(s.ctx, params)
	s.Require().NoError(err)

	fcAddr := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	burnErr := errors.New("burn coins failed")
	burnCoins := sdk.NewCoins(sdk.NewCoin("alyth", math.NewInt(50)))

	gomock.InOrder(
		s.bankKeeper.EXPECT().GetBalance(gomock.Any(), fcAddr, "alyth").Return(sdk.NewCoin("alyth", math.NewInt(100))),
		s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), authtypes.FeeCollectorName, "burn", burnCoins).Return(nil),
		s.bankKeeper.EXPECT().BurnCoins(gomock.Any(), "burn", burnCoins).Return(burnErr),
	)

	err = s.burnKeeper.ProcessFeeBurn(s.ctx)
	s.Require().Error(err)
	s.Require().ErrorIs(err, burnErr)
}
