package server

import (
	"github.com/xavier-niu/xnrpc/pkg/rpc/codec"
	"reflect"
)

type request struct {
	h            *codec.Header // method
	argv, replyv reflect.Value // arg and reply
}
