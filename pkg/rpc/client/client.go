package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xavier-niu/xnrpc/pkg/rpc"
	"github.com/xavier-niu/xnrpc/pkg/rpc/codec"
	"io"
	"log"
	"net"
	"sync"
)

var ErrShutdown = errors.New("connection has been shut down")

// ----- Client ------

type Client struct {
	cc       codec.Codec
	opt      *rpc.Option
	sending  sync.Mutex // deliver requests orderly
	header   codec.Header
	mu       sync.Mutex // protect following variables
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // close by user
	shutdown bool // errors occurred
}

// Client should conform to io.Closer
var _ io.Closer = (*Client)(nil)

func NewClient(conn net.Conn, opt *rpc.Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}
	// communicate with server for option
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error: ", err)
		_ = conn.Close()
		return nil, err
	}
	client := &Client{
		cc: f(conn),
		opt: opt,
		seq: 1,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client, nil
}

func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

func (client *Client) terminatesCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

func (client *Client) receive() {
	var err error
	// if there is no error, client will receive the message from server forever.
	for err == nil {
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		case call == nil:
			// h.Seq is not existed
			err = client.cc.ReadBody(nil)
		case h.Error != "":
			// h.Seq is valid but client reports an error
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			// h.Seq is valid and client returns expectedly
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body: " + err.Error())
			}
			call.done()
		}
	}
	// errors occurred, terminates all calls
	client.terminatesCalls(err)
}

func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()

	// register the call
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	// request header
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	// send the request
	if err := client.cc.Write(&client.header, call.Args); err != nil {
		// handle request error
		call := client.removeCall(seq)
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// Go send call request asynchronously by channel
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		// TODO: why here needs a channel with 10 buffers?
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args: args,
		Reply: reply,
		Done: done,
	}
	client.send(call)
	return call
}

func (client *Client) Call(serverMethod string, args, reply interface{}) error {
	call := <- client.Go(serverMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}

// ----- Dial -----

func parseOptions(opts ...*rpc.Option) (*rpc.Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return rpc.DefaultOption, nil
	}
	if len(opts) > 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = rpc.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = rpc.DefaultOption.CodecType
	}
	return opt, nil
}

func Dial(network, address string, opts ...*rpc.Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return
	}

	conn, err := net.Dial(network, address)
	if err != nil {
		return
	}
	defer func() {
		// close conn if client is not created
		if client == nil { _ = conn.Close() }
	}()

	return NewClient(conn, opt)
}
