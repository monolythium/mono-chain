package validator_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	validatortypes "github.com/monolythium/mono-chain/x/validator/types"
)

// routerExec dispatches msg through the MsgServiceRouter with the fixture's
// context. This exercises the circuit breaker (unlike direct keeper calls).
func routerExec(f *fixture, msg sdk.Msg) (*sdk.Result, error) {
	handler := f.app.MsgServiceRouter().Handler(msg)
	if handler == nil {
		return nil, fmt.Errorf("no handler for %T", msg)
	}
	return handler(f.sdkCtx, msg)
}

func newCreateValidatorMsg(t *testing.T, valAddr sdk.ValAddress) *stakingtypes.MsgCreateValidator {
	t.Helper()
	pubKey := ed25519.GenPrivKey().PubKey()
	msg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		pubKey,
		sdk.NewCoin(sdk.DefaultBondDenom, sdk.DefaultPowerReduction),
		stakingtypes.Description{Moniker: "test-val"},
		stakingtypes.CommissionRates{
			Rate:          math.LegacyNewDecWithPrec(10, 2),
			MaxRate:       math.LegacyNewDecWithPrec(20, 2),
			MaxChangeRate: math.LegacyNewDecWithPrec(1, 2),
		},
		math.OneInt(),
	)
	require.NoError(t, err)
	return msg
}

func TestCircuitBreaker_BlocksMsgCreateValidator(t *testing.T) {
	f := initFixture(t)
	f.sdkCtx = f.sdkCtx.WithBlockHeight(1)

	addrs := simtestutil.AddTestAddrsIncremental(f.bankKeeper, f.stakingKeeper, f.sdkCtx, 1,
		sdk.DefaultPowerReduction.MulRaw(100))
	valAddr := sdk.ValAddress(addrs[0])

	_, err := routerExec(f, newCreateValidatorMsg(t, valAddr))
	require.Error(t, err)
	require.Contains(t, err.Error(), "circuit breaker")
}

func TestCircuitBreaker_AllowsMsgCreateValidatorAtGenesis(t *testing.T) {
	f := initFixture(t)

	addrs := simtestutil.AddTestAddrsIncremental(f.bankKeeper, f.stakingKeeper, f.sdkCtx, 1,
		sdk.DefaultPowerReduction.MulRaw(100))
	valAddr := sdk.ValAddress(addrs[0])

	_, err := routerExec(f, newCreateValidatorMsg(t, valAddr))
	require.NoError(t, err)

	val, err := f.stakingKeeper.GetValidator(f.sdkCtx, valAddr)
	require.NoError(t, err)
	require.Equal(t, valAddr.String(), val.OperatorAddress)
}

func TestCircuitBreaker_AllowsMsgRegisterValidator(t *testing.T) {
	f := initFixture(t)
	f.sdkCtx = f.sdkCtx.WithBlockHeight(1)

	regBurnAmt := math.NewIntWithDecimal(100_000, 18)
	minSelfDelAmt := math.NewIntWithDecimal(100_000, 18)
	totalNeeded := regBurnAmt.Add(minSelfDelAmt)

	addrs := simtestutil.AddTestAddrsIncremental(f.bankKeeper, f.stakingKeeper, f.sdkCtx, 1, totalNeeded)
	valAddr := sdk.ValAddress(addrs[0])
	pubKey := ed25519.GenPrivKey().PubKey()

	createValMsg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		pubKey,
		sdk.NewCoin(sdk.DefaultBondDenom, minSelfDelAmt),
		stakingtypes.Description{Moniker: "registered-val"},
		stakingtypes.CommissionRates{
			Rate:          math.LegacyNewDecWithPrec(10, 2),
			MaxRate:       math.LegacyNewDecWithPrec(20, 2),
			MaxChangeRate: math.LegacyNewDecWithPrec(1, 2),
		},
		minSelfDelAmt,
	)
	require.NoError(t, err)

	registerMsg := &validatortypes.MsgRegisterValidator{
		Sender:          addrs[0].String(),
		CreateValidator: *createValMsg,
		Burn:            sdk.NewCoin(sdk.DefaultBondDenom, regBurnAmt),
	}

	_, err = routerExec(f, registerMsg)
	require.NoError(t, err, "MsgRegisterValidator must not be blocked by circuit breaker")

	val, err := f.stakingKeeper.GetValidator(f.sdkCtx, valAddr)
	require.NoError(t, err)
	require.Equal(t, valAddr.String(), val.OperatorAddress)
}
