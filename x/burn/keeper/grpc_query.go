package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/monolythium/mono-chain/x/burn/types"
)

var _ types.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the QueryServer interface
func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{k}
}

type queryServer struct {
	k Keeper
}

func (q queryServer) Params(
	ctx context.Context,
	_ *types.QueryParamsRequest,
) (*types.QueryParamsResponse, error) {
	params, err := q.k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: params}, nil
}

func (q queryServer) BurnStats(
	ctx context.Context,
	_ *types.QueryBurnStatsRequest,
) (*types.QueryBurnStatsResponse, error) {
	count, err := q.k.GlobalBurnCount.Peek(ctx)
	if err != nil {
		return nil, err
	}

	globalBurnTotal, err := q.k.GetGlobalBurnTotal(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryBurnStatsResponse{
		TotalBurns:  count,
		TotalBurned: globalBurnTotal.String(),
	}, nil
}

func (q queryServer) AccountBurns(
	ctx context.Context,
	req *types.QueryAccountBurnsRequest,
) (*types.QueryAccountBurnsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addrBytes, err := q.k.addressCodec.StringToBytes(req.Address)
	if err != nil {
		return nil, err
	}

	accountBurnTotal, err := q.k.GetAccountBurnTotal(ctx, sdk.AccAddress(addrBytes))
	if err != nil {
		return nil, err
	}

	return &types.QueryAccountBurnsResponse{
		Amount: accountBurnTotal.String(),
	}, nil
}
