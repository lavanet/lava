package thirdparty_utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/lavanet/lava/utils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// this was an attempt to create a generic callback to save some code from the implemented rpc's but
// because its a type specific interface result, we dont get anything from that.

type ThirdPartyGeneticCallBackCaller struct {
	CallBack func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
}

func (qs *ThirdPartyGeneticCallBackCaller) CallbackCaller(ctx context.Context, method string, requestType interface{}, responseType interface{}) (interface{}, error) {
	reqMarshaled, err := json.Marshal(requestType)
	log.Println("json: ", reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err, nil)
	}
	res, err := qs.CallBack(ctx, method, reqMarshaled)
	if err != nil {
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err, nil)
	}
	switch t := responseType.(type) {
	case protoreflect.ProtoMessage:
		err = proto.Unmarshal(res, t)
		if err != nil {
			return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err, nil)
		}
		return t, nil
	default:
		return nil, utils.LavaFormatError("Unsupported interface type", nil, &map[string]string{"type": fmt.Sprintf("%T", t)})
	}
}
