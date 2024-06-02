// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpcclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"

	"github.com/ethereum/go-ethereum/log"
	"github.com/lavanet/lava/utils"
)

var (
	ErrClientQuit                = errors.New("client is closed")
	ErrSubscriptionQueueOverflow = errors.New("subscription queue overflow")
	errClientReconnected         = errors.New("client reconnected")
	errDead                      = errors.New("connection lost")
)

const (
	// Timeouts
	defaultDialTimeout = 10 * time.Second // used if context has no deadline
	subscribeTimeout   = 5 * time.Second  // overall timeout eth_subscribe, rpc_modules calls
)

const (
	// Subscriptions are removed when the subscriber cannot keep up.
	//
	// This can be worked around by supplying a channel with sufficiently sized buffer,
	// but this can be inconvenient and hard to explain in the docs. Another issue with
	// buffered channels is that the buffer is static even though it might not be needed
	// most of the time.
	//
	// The approach taken here is to maintain a per-subscription linked list buffer
	// shrinks on demand. If the buffer reaches the size below, the subscription is
	// dropped.
	maxClientSubscriptionBuffer = 20000
)

// BatchElem is an element in a batch request.
type BatchElemWithId struct {
	Method string
	args   interface{}
	// The result is unmarshaled into this field. Result must be set to a
	// non-nil pointer value of the desired type, otherwise the response will be
	// discarded.
	Result interface{}
	// Error is set if the server returns an error for this request, or if
	// unmarshaling into Result fails. It is not set for I/O errors.
	Error error
	ID    json.RawMessage // added an ID field because we build our messages with built in ID, this is optional and can be set to nil
}

func NewBatchElementWithId(method string, args interface{}, result interface{}, iD json.RawMessage) (BatchElemWithId, error) {
	_, ok := args.([]interface{})
	_, ok2 := args.(map[string]interface{})
	if !ok && !ok2 && args != nil {
		return BatchElemWithId{}, fmt.Errorf("invalid args supported types are []interface{} or map[string]interface{}")
	}
	return BatchElemWithId{
		Method: method,
		args:   args,
		Result: result,
		ID:     iD,
	}, nil
}

// Client represents a connection to an RPC server.
type Client struct {
	idgen    func() ID // for subscriptions
	isHTTP   bool      // connection type: http, ws or ipc
	services *serviceRegistry

	idCounter uint32

	// This function, if non-nil, is called when the connection is lost.
	reconnectFunc reconnectFunc

	// writeConn is used for writing to the connection on the caller's goroutine. It should
	// only be accessed outside of dispatch, with the write lock held. The write lock is
	// taken by sending on reqInit and released by sending on reqSent.
	writeConn jsonWriter

	// for dispatch
	close       chan struct{}
	closing     chan struct{}    // closed when client is quitting
	didClose    chan struct{}    // closed when client quits
	reconnected chan ServerCodec // where write/reconnect sends the new connection
	readOp      chan readOp      // read messages
	readErr     chan error       // errors from read
	reqInit     chan *requestOp  // register response IDs, takes write lock
	reqSent     chan error       // signals write completion, releases write lock
	reqTimeout  chan *requestOp  // removes response IDs when call timeout expires
}

type reconnectFunc func(ctx context.Context) (ServerCodec, error)

type clientContextKey struct{}

type clientConn struct {
	codec   ServerCodec
	handler *handler
}

func (c *Client) newClientConn(conn ServerCodec) *clientConn {
	ctx := context.Background()
	ctx = context.WithValue(ctx, clientContextKey{}, c)
	ctx = context.WithValue(ctx, peerInfoContextKey{}, conn.peerInfo())
	handler := newHandler(ctx, conn, c.idgen, c.services)
	return &clientConn{conn, handler}
}

func (cc *clientConn) close(err error, inflightReq *requestOp) {
	cc.handler.close(err, inflightReq)
	cc.codec.close()
}

type readOp struct {
	msgs  []*JsonrpcMessage
	batch bool
}

type requestOp struct {
	ids   []json.RawMessage
	err   error
	resp  chan *JsonrpcMessage // receives up to len(ids) responses
	sub   *ClientSubscription  // only set for EthSubscribe requests
	subId string               // this is only set for tendermint subscription
}

func (op *requestOp) wait(ctx context.Context, c *Client) (*JsonrpcMessage, error) {
	select {
	case <-ctx.Done():
		// Send the timeout to dispatch so it can remove the request IDs.
		if !c.isHTTP {
			select {
			case c.reqTimeout <- op:
			case <-c.closing:
			}
		}
		return nil, ctx.Err()
	case resp := <-op.resp:
		return resp, op.err
	}
}

// Dial creates a new client for the given URL.
//
// The currently supported URL schemes are "http", "https", "ws" and "wss". If rawurl is a
// file name with no URL scheme, a local socket connection is established using UNIX
// domain sockets on supported platforms and named pipes on Windows. If you want to
// configure transport options, use DialHTTP, DialWebsocket or DialIPC instead.
//
// For websocket connections, the origin is set to the local host name.
//
// The client reconnects automatically if the connection is lost.
func Dial(rawurl string) (*Client, error) {
	return DialContext(context.Background(), rawurl)
}

// DialContext creates a new RPC client, just like Dial.
//
// The context is used to cancel or time out the initial connection establishment. It does
// not affect subsequent interactions with the client.
func DialContext(ctx context.Context, rawurl string) (*Client, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https":
		return DialHTTP(rawurl)
	case "ws", "wss":
		return DialWebsocket(ctx, rawurl, "")
	case "stdio":
		return DialStdIO(ctx)
	case "":
		return DialIPC(ctx, rawurl)
	default:
		return nil, fmt.Errorf("no known transport for URL scheme %q", u.Scheme)
	}
}

// ClientFromContext retrieves the client from the context, if any. This can be used to perform
// 'reverse calls' in a handler method.
func ClientFromContext(ctx context.Context) (*Client, bool) {
	client, ok := ctx.Value(clientContextKey{}).(*Client)
	return client, ok
}

func newClient(initctx context.Context, connect reconnectFunc) (*Client, error) {
	conn, err := connect(initctx)
	if err != nil {
		return nil, err
	}
	c := initClient(conn, randomIDGenerator(), new(serviceRegistry))
	c.reconnectFunc = connect
	return c, nil
}

func initClient(conn ServerCodec, idgen func() ID, services *serviceRegistry) *Client {
	_, isHTTP := conn.(*httpConn)
	c := &Client{
		isHTTP:      isHTTP,
		idgen:       idgen,
		services:    services,
		writeConn:   conn,
		close:       make(chan struct{}),
		closing:     make(chan struct{}),
		didClose:    make(chan struct{}),
		reconnected: make(chan ServerCodec),
		readOp:      make(chan readOp),
		readErr:     make(chan error),
		reqInit:     make(chan *requestOp),
		reqSent:     make(chan error, 1),
		reqTimeout:  make(chan *requestOp),
	}
	if !isHTTP {
		go c.dispatch(conn)
	}
	return c
}

// RegisterName creates a service for the given receiver type under the given name. When no
// methods on the given receiver match the criteria to be either a RPC method or a
// subscription an error is returned. Otherwise a new service is created and added to the
// service collection this client provides to the server.
func (c *Client) RegisterName(name string, receiver interface{}) error {
	return c.services.registerName(name, receiver)
}

func (c *Client) nextID() json.RawMessage {
	id := atomic.AddUint32(&c.idCounter, 1)
	return strconv.AppendUint(nil, uint64(id), 10)
}

// Close closes the client, aborting any in-flight requests.
func (c *Client) Close() {
	if c.isHTTP {
		return
	}
	select {
	case c.close <- struct{}{}:
		<-c.didClose
	case <-c.didClose:
	}
}

// SetHeader adds a custom HTTP header to the client's requests.
// This method only works for clients using HTTP, it doesn't have
// any effect for clients using another transport.
func (c *Client) SetHeader(key, value string) {
	if !c.isHTTP {
		return
	}
	conn, ok := c.writeConn.(*httpConn)
	if !ok {
		panic("SetHeader - c.writeConn.(*httpConn) - type assertion failed")
	}
	conn.mu.Lock()
	conn.headers.Set(key, value)
	conn.mu.Unlock()
}

// CallContext performs a JSON-RPC call with the given arguments. If the context is
// canceled before the call has successfully returned, CallContext returns immediately.
//
// The result must be a pointer so that package json can unmarshal into it. You
// can also pass nil, in which case the result is ignored.
func (c *Client) CallContext(ctx context.Context, id json.RawMessage, method string, params interface{}, isJsonRPC bool, strict bool) (*JsonrpcMessage, error) {
	var msg *JsonrpcMessage
	var err error
	switch p := params.(type) {
	case []interface{}, string:
		msg, err = c.newMessageArrayWithID(method, id, p)
	case map[string]interface{}:
		msg, err = c.newMessageMapWithID(method, id, p)
	case nil:
		msg, err = c.newMessageArrayWithID(method, id, (make([]interface{}, 0))) // in case of nil, we will send it as an empty array.
	default:
		return nil, fmt.Errorf("%s unknown parameters type %s", p, reflect.TypeOf(p))
	}

	if err != nil {
		return nil, err
	}

	op := &requestOp{ids: []json.RawMessage{msg.ID}, resp: make(chan *JsonrpcMessage, 1)}

	if c.isHTTP {
		err = c.sendHTTP(ctx, op, msg, isJsonRPC, strict)
	} else {
		err = c.send(ctx, op, msg)
	}
	if err != nil {
		return nil, err
	}

	// dispatch has accepted the request and will close the channel when it quits.

	resp, err := op.wait(ctx, c)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// BatchCall sends all given requests as a single batch and waits for the server
// to return a response for all of them.
//
// In contrast to Call, BatchCall only returns I/O errors. Any error specific to
// a request is reported through the Error field of the corresponding BatchElem.
//
// Note that batch calls may not be executed atomically on the server side.
func (c *Client) BatchCall(b []BatchElemWithId, strict bool) error {
	ctx := context.Background()
	return c.BatchCallContext(ctx, b, strict)
}

// BatchCallContext sends all given requests as a single batch and waits for the server
// to return a response for all of them. The wait duration is bounded by the
// context's deadline.
//
// In contrast to CallContext, BatchCallContext only returns errors that have occurred
// while sending the request. Any error specific to a request is reported through the
// Error field of the corresponding BatchElem.
//
// Note that batch calls may not be executed atomically on the server side.
func (c *Client) BatchCallContext(ctx context.Context, b []BatchElemWithId, strict bool) error {
	var (
		msgs = make([]*JsonrpcMessage, len(b))
		byID = make(map[string]int, len(b))
	)
	op := &requestOp{
		ids:  make([]json.RawMessage, len(b)),
		resp: make(chan *JsonrpcMessage, len(b)),
	}
	for i, elem := range b {
		var msg *JsonrpcMessage
		var err error
		switch args := elem.args.(type) {
		case []interface{}:
			msg, err = c.newMessageArray(elem.Method, args...)
		case map[string]interface{}:
			msg, err = c.newMessageMapWithID(elem.Method, elem.ID, args)
		case nil:
			msg, err = c.newMessageArray(elem.Method) // in case of nil, we will send it as an empty array.
		default:
			return fmt.Errorf("invalid args type in message %t", args)
		}

		if err != nil {
			return err
		}
		if elem.ID != nil {
			msg.ID = elem.ID
		}
		_, exists := byID[string(msg.ID)]
		if exists {
			nextID := c.nextID()
			for _, exists := byID[string(nextID)]; exists; _, exists = byID[string(nextID)] {
				nextID = c.nextID() // in case the user specified something nextID also tries to send
			}
			msg.ID = nextID
			byID[string(msg.ID)] = i
		} else {
			byID[string(msg.ID)] = i
		}
		msgs[i] = msg
		op.ids[i] = msg.ID
	}
	timeToSend := time.Duration(0)
	deadline, ok := ctx.Deadline()
	if ok {
		timeToSend = time.Until(deadline)
	}
	var err error
	if c.isHTTP {
		err = c.sendBatchHTTP(ctx, op, msgs, strict)
	} else {
		err = c.send(ctx, op, msgs)
	}

	// Wait for all responses to come back.
	n := 0
	for ; n < len(b) && err == nil; n++ {
		var resp *JsonrpcMessage
		resp, err = op.wait(ctx, c)
		if err != nil {
			break
		}
		// Find the element corresponding to this response.
		// The element is guaranteed to be present because dispatch
		// only sends valid IDs to our channel.
		byOrder, found := byID[string(resp.ID)]
		if !found {
			utils.LavaFormatError("invalid ID returned", nil, utils.LogAttr("id", string(resp.ID)))
		}
		elem := &b[byOrder]
		if resp.Error != nil {
			elem.Error = resp.Error
			continue
		}
		elem.Error = json.Unmarshal(resp.Result, elem.Result)
	}
	// when timing out, return extra data on partial ids support
	if err != nil && n > 0 && n < len(b) {
		validIds := []json.RawMessage{}
		missingIds := []json.RawMessage{}
		for _, elem := range b {
			if elem.Error == nil && elem.Result == nil {
				missingIds = append(missingIds, elem.ID)
			} else {
				validIds = append(validIds, elem.ID)
			}
		}
		return utils.LavaFormatError("partial batches response", err, utils.LogAttr("timeToProcess", timeToSend), utils.LogAttr("supported", n), utils.LogAttr("valid_ids", validIds), utils.LogAttr("missing_ids", missingIds))
	}
	return err
}

// Notify sends a notification, i.e. a method call that doesn't expect a response.
func (c *Client) Notify(ctx context.Context, method string, args ...interface{}) error {
	op := new(requestOp)
	msg, err := c.newMessageArray(method, args...)
	if err != nil {
		return err
	}
	msg.ID = nil

	if c.isHTTP {
		return c.sendHTTP(ctx, op, msg, true, false)
	}
	return c.send(ctx, op, msg)
}

// Subscribe calls the "<namespace>_subscribe" method with the given arguments,
// registering a subscription. Server notifications for the subscription are
// sent to the given channel. The element type of the channel must match the
// expected type of content returned by the subscription.
//
// The context argument cancels the RPC request that sets up the subscription but has no
// effect on the subscription after Subscribe has returned.
//
// Slow subscribers will be dropped eventually. Client buffers up to 20000 notifications
// before considering the subscriber dead. The subscription Err channel will receive
// ErrSubscriptionQueueOverflow. Use a sufficiently large buffer on the channel or ensure
// that the channel usually has at least one reader to prevent this issue.
func (c *Client) Subscribe(ctx context.Context, id json.RawMessage, method string, channel, params interface{}) (*ClientSubscription, *JsonrpcMessage, error) {
	// Check type of channel first.
	chanVal := reflect.ValueOf(channel)
	if chanVal.Kind() != reflect.Chan || chanVal.Type().ChanDir()&reflect.SendDir == 0 {
		panic(fmt.Sprintf("channel argument of Subscribe has type %T, need writable channel", channel))
	}
	if chanVal.IsNil() {
		panic("channel given to Subscribe must not be nil")
	}
	if c.isHTTP {
		return nil, nil, ErrNotificationsUnsupported
	}
	var msg *JsonrpcMessage
	var err error
	var subId string
	var ok bool
	switch p := params.(type) {
	case []interface{}:
		msg, err = c.newMessageArrayWithID(method, id, p)
	case map[string]interface{}:
		msg, err = c.newMessageMapWithID(method, id, p)
		subId, ok = p["query"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("Subscribe - p['query'].(string) - type assertion failed")
		}
	default:
		return nil, nil, fmt.Errorf("%s unknown parameters type %s", p, reflect.TypeOf(p))
	}
	if err != nil {
		return nil, nil, err
	}

	op := &requestOp{
		ids:   []json.RawMessage{msg.ID},
		resp:  make(chan *JsonrpcMessage),
		sub:   newClientSubscription(c, method, chanVal),
		subId: subId,
	}

	// Send the subscription request.
	// The arrival and validity of the response is signaled on sub.quit.
	if err := c.send(ctx, op, msg); err != nil {
		return nil, nil, err
	}
	resp, err := op.wait(ctx, c)
	// In the case of response containing the error message, we want to return it to the user as-is
	if err != nil && resp != nil && resp.Error != nil {
		return nil, resp, nil
	}

	if err != nil {
		return nil, nil, err
	}

	return op.sub, resp, nil
}

func (c *Client) newMessageArrayWithID(method string, id json.RawMessage, paramsIn interface{}) (*JsonrpcMessage, error) {
	var msg *JsonrpcMessage
	if id == nil {
		msg = &JsonrpcMessage{Version: Vsn, ID: c.nextID(), Method: method}
	} else {
		msg = &JsonrpcMessage{Version: Vsn, ID: id, Method: method}
	}
	if paramsIn != nil { // prevent sending "params":null
		var err error
		if msg.Params, err = json.Marshal(paramsIn); err != nil {
			return nil, err
		}
	}
	return msg, nil
}

func (c *Client) newMessageArray(method string, paramsIn ...interface{}) (*JsonrpcMessage, error) {
	msg := &JsonrpcMessage{Version: Vsn, ID: c.nextID(), Method: method}
	if paramsIn != nil { // prevent sending "params":null
		var err error
		if msg.Params, err = json.Marshal(paramsIn); err != nil {
			return nil, err
		}
	}
	return msg, nil
}

func (c *Client) newMessageMapWithID(method string, id json.RawMessage, paramsIn map[string]interface{}) (*JsonrpcMessage, error) {
	var msg *JsonrpcMessage
	if id == nil {
		msg = &JsonrpcMessage{Version: Vsn, ID: c.nextID(), Method: method}
	} else {
		msg = &JsonrpcMessage{Version: Vsn, ID: id, Method: method}
	}
	if paramsIn != nil { // prevent sending "params":null
		var err error
		if msg.Params, err = json.Marshal(paramsIn); err != nil {
			return nil, err
		}

		// test this too:
		//
		// var paramsMap = make(map[string]json.RawMessage, len(paramsIn))
		// for name, value := range paramsIn {
		// 	valueJSON, err := json.Marshal(value)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	paramsMap[name] = valueJSON
		// }
		// if msg.Params, err = json.Marshal(paramsMap); err != nil {
		// 	return nil, err
		// }
	}
	return msg, nil
}

// send registers op with the dispatch loop, then sends msg on the connection.
// if sending fails, op is deregistered.
func (c *Client) send(ctx context.Context, op *requestOp, msg interface{}) error {
	select {
	case c.reqInit <- op:
		err := c.write(ctx, msg, false)
		c.reqSent <- err
		return err
	case <-ctx.Done():
		// This can happen if the client is overloaded or unable to keep up with
		// subscription notifications.
		return ctx.Err()
	case <-c.closing:
		return ErrClientQuit
	}
}

func (c *Client) write(ctx context.Context, msg interface{}, retry bool) error {
	if c.writeConn == nil {
		// The previous write failed. Try to establish a new connection.
		if err := c.reconnect(ctx); err != nil {
			return err
		}
	}
	err := c.writeConn.writeJSON(ctx, msg)
	if err != nil {
		c.writeConn = nil
		if !retry {
			return c.write(ctx, msg, true)
		}
	}
	return err
}

func (c *Client) reconnect(ctx context.Context) error {
	if c.reconnectFunc == nil {
		return errDead
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, defaultDialTimeout)
		defer cancel()
	}
	newconn, err := c.reconnectFunc(ctx)
	if err != nil {
		log.Trace("RPC client reconnect failed", "err", err)
		return err
	}
	select {
	case c.reconnected <- newconn:
		c.writeConn = newconn
		return nil
	case <-c.didClose:
		newconn.close()
		return ErrClientQuit
	}
}

// dispatch is the main loop of the client.
// It sends read messages to waiting calls to Call and BatchCall
// and subscription notifications to registered subscriptions.
func (c *Client) dispatch(codec ServerCodec) {
	var (
		lastOp      *requestOp  // tracks last send operation
		reqInitLock = c.reqInit // nil while the send lock is held
		conn        = c.newClientConn(codec)
		reading     = true
	)
	defer func() {
		close(c.closing)
		if reading {
			conn.close(ErrClientQuit, nil)
			c.drainRead()
		}
		close(c.didClose)
	}()

	// Spawn the initial read loop.
	go c.read(codec)

	for {
		select {
		case <-c.close:
			return

		// Read path:
		case op := <-c.readOp:
			if op.batch {
				conn.handler.handleBatch(op.msgs)
			} else {
				conn.handler.handleMsg(op.msgs[0])
			}

		case err := <-c.readErr:
			conn.handler.log.Debug("RPC connection read error", "err", err)
			conn.close(err, lastOp)
			reading = false

		// Reconnect:
		case newcodec := <-c.reconnected:
			log.Debug("RPC client reconnected", "reading", reading, "conn", newcodec.remoteAddr())
			if reading {
				// Wait for the previous read loop to exit. This is a rare case which
				// happens if this loop isn't notified in time after the connection breaks.
				// In those cases the caller will notice first and reconnect. Closing the
				// handler terminates all waiting requests (closing op.resp) except for
				// lastOp, which will be transferred to the new handler.
				conn.close(errClientReconnected, lastOp)
				c.drainRead()
			}
			go c.read(newcodec)
			reading = true
			conn = c.newClientConn(newcodec)
			// Re-register the in-flight request on the new handler
			// because that's where it will be sent.
			conn.handler.addRequestOp(lastOp)

		// Send path:
		case op := <-reqInitLock:
			// Stop listening for further requests until the current one has been sent.
			reqInitLock = nil
			lastOp = op
			conn.handler.addRequestOp(op)

		case err := <-c.reqSent:
			if err != nil {
				// Remove response handlers for the last send. When the read loop
				// goes down, it will signal all other current operations.
				conn.handler.removeRequestOp(lastOp)
			}
			// Let the next request in.
			reqInitLock = c.reqInit
			lastOp = nil

		case op := <-c.reqTimeout:
			conn.handler.removeRequestOp(op)
		}
	}
}

// drainRead drops read messages until an error occurs.
func (c *Client) drainRead() {
	for {
		select {
		case <-c.readOp:
		case <-c.readErr:
			return
		}
	}
}

// read decodes RPC messages from a codec, feeding them into dispatch.
func (c *Client) read(codec ServerCodec) {
	for {
		msgs, batch, err := codec.readBatch()
		if _, ok := err.(*json.SyntaxError); ok {
			codec.writeJSON(context.Background(), errorMessage(&parseError{err.Error()}))
		}
		if err != nil {
			c.readErr <- err
			return
		}
		c.readOp <- readOp{msgs, batch}
	}
}
