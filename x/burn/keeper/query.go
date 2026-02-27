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

func (q queryServer) BurnStats(ctx context.Context, req *types.QueryBurnStatsRequest) (*types.QueryBurnStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	count, err := q.k.BurnCount.Peek(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	has, err := q.k.BurnTotal.Has(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	var total sdk.Coin
	if has {
		total, err = q.k.BurnTotal.Get(ctx)
		if err != nil {
			return nil, status.Error(codes.Internal, "internal error")
		}
	} else {
		total = sdk.Coin{}
	}

	return &types.QueryBurnStatsResponse{
		TotalBurns:  count,
		TotalBurned: total.String(),
	}, nil
}

func (q queryServer) AccountBurns(ctx context.Context, req *types.QueryAccountBurnsRequest) (*types.QueryAccountBurnsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	has, err := q.k.BurnAccountTotal.Has(ctx, addr)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	var amount sdk.Coin
	if has {
		amount, err = q.k.BurnAccountTotal.Get(ctx, addr)
		if err != nil {
			return nil, status.Error(codes.Internal, "internal error")
		}
	} else {
		amount = sdk.Coin{}
	}

	return &types.QueryAccountBurnsResponse{
		Amount: amount.String(),
	}, nil
}
