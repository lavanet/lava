package main

import (
	"bytes"
	"context"
	"log"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/lavanet/lava/relayer/chainproxy/grpcutil"
)

const addr = "127.0.0.1:3342"

type GetLatestBlockResponseWrapper tmservice.GetLatestBlockResponse

func (m *GetLatestBlockResponseWrapper) Reset() {
	v := tmservice.GetLatestBlockResponse(*m)
	v.Reset()
	*m = GetLatestBlockResponseWrapper(v)
}

func (m *GetLatestBlockResponseWrapper) String() string {
	v := tmservice.GetLatestBlockResponse(*m)
	return v.String()
}

func (m *GetLatestBlockResponseWrapper) ProtoMessage() {
	v := tmservice.GetLatestBlockResponse(*m)
	v.ProtoMessage()
}

func (m *GetLatestBlockResponseWrapper) Unmarshal(dAtA []byte) error {
	t := 0
	from, to := -1, -1
	for i, b := range dAtA {
		switch b {
		case byte('{'):
			t++
			if from == -1 {
				from = i
			}
		case byte('}'):
			t--
			to = i
		default:
			continue
		}
		if t == 0 && from != -1 {
			break
		}
	}
	if from == -1 || !(to > from) {
		return nil
	}
	return jsonpb.Unmarshal(bytes.NewReader(dAtA[from:to+1]), m)
}

type WrapPR struct {
	proto.Message
}

func (w *WrapPR) Unmarshal(dAtA []byte) error {
	return nil
}

func main() {
	asd()
	// _ = callGetLatestBlockRequest()
}

func callGetLatestBlockRequest() (err error) {
	ctx := context.Background()
	conn := grpcutil.MustDial(ctx, addr)
	defer func() { _ = conn.Close() }()

	// method := "/cosmos.base.tendermint.v1beta1.Service/GetLatestBlock"
	req := &tmservice.GetLatestBlockRequest{}

	cl := tmservice.NewServiceClient(conn)
	res, err := cl.GetLatestBlock(ctx, req)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// var res GetLatestBlockResponseWrapper
	// if err = conn.Invoke(ctx, method, req, &res); err != nil {
	// 	log.Println(err.Error())
	// 	return err
	// }

	log.Println(res.String())

	// log.Println(res.String())
	return nil
}

func asd() {
	var resp tmservice.GetLatestBlockResponse
	if err := resp.Unmarshal([]byte(dd)); err != nil {
		log.Println(err.Error())
		return
	}
	// b := findJSON([]byte(dd))
	// if err := json.Unmarshal(b, &resp); err != nil {
	// 	log.Println(err.Error())
	// 	return
	// }
	log.Println(resp.String())
}

var dd = "H\n �����;�weh/�� ;Ag��e)$�\u001EeVyKh,�\u001F\u0012$\b\u0001\u0012 \t�DL�\u000FW�\u001B�\u0001v�?�N�Ѹj�[�;>��9\u0018�\u001E�\u0012�\u0004\n�\u0003\n\u0002\b\v\u0012\u0004lava\u0018>\"\f\b���\u0006\u0010����\u0002*H\n �\u0001�\u001F/�B�(�H�\u001E\u0003^��Ebr%��Ϯ���z\uAB27\u0012$\b\u0001\u0012 �\u001AH�-��+���\u00151��ƍ\u001ES�h\n+�+p��9(�2 ��\u0002˸\u0014��\u0017,^X%���\u001C�zb\u0003\"^\u0003�r���U\u0002�: ��B��\u001C\u0014���șo�$'�A�d��L���\u001BxR�UB \a�ʰ��{��r�:���a蹉0��<�\akr\v1\u0002�J \a�ʰ��{��r�:���a蹉0��<�\akr\v1\u0002�R \u0004���}�(?w����<D�X�ߊ���t\u0005ط�ڭ�/Z �\t6z�\u000E�#\n���MZ��\u001B7\u0012��\u0002'�Y�\u000F�ˎׂb ��B��\u001C\u0014���șo�$'�A�d��L���\u001BxR�Uj ��B��\u001C\u0014���șo�$'�A�d��L���\u001BxR�Ur\u0014����\\���G�]��lZ�(\u000F�\u0012 \u001A \"�\u0001\b=\u001AH\n �\u0001�\u001F/�B�(�H�\u001E\u0003^��Ebr%��Ϯ���z\uAB27\u0012$\b\u0001\u0012 �\u001AH�-��+���\u00151��ƍ\u001ES�h\n+�+p��9(�\"h\b\u0002\u0012\u0014����\\���G�]��lZ�(\u000F�\u001A\f\b���\u0006\u0010����\u0002\"@\f���z\u001DJ�{�줌����-\u0010�����|F��ä���\u001EKJt$5\a��\u001Ecb-^�\u0012E���\u001F\u001B�\u001Be�8\u000E"

func findJSON(dAtA []byte) []byte {
	t := 0
	from, to := -1, -1
	for i, b := range dAtA {
		switch b {
		case byte('{'):
			t++
			if from == -1 {
				from = i
			}
		case byte('}'):
			t--
			to = i
		default:
			continue
		}
		if t == 0 && from != -1 {
			break
		}
	}
	if from == -1 || !(to > from) {
		return dAtA
	}
	return dAtA[from:to]
}
