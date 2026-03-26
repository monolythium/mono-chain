package testutil

import (
	"errors"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/gogoproto/proto"
)

// FailMarshalCodec wraps a real codec but errors on MarshalJSON.
type FailMarshalCodec struct{ codec.Codec }

func (FailMarshalCodec) MarshalJSON(proto.Message) ([]byte, error) {
	return nil, errors.New("marshal failed")
}
