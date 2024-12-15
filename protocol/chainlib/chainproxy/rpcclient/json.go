// Copyright 2015 The go-ethereum Authors
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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"

	"github.com/lavanet/lava/v4/utils"
)

const (
	Vsn                              = "2.0"
	serviceMethodSeparator           = "_"
	subscribeMethodSuffix            = "_subscribe"
	unsubscribeMethodSuffix          = "_unsubscribe"
	ethereumNotificationMethodSuffix = "_subscription"
	solanaNotificationMethodSuffix   = "Notification"

	defaultWriteTimeout = 10 * time.Second // used if context has no deadline
)

var null = json.RawMessage("null")

type ethereumSubscriptionResult struct {
	ID     string          `json:"subscription"`
	Result json.RawMessage `json:"result,omitempty"`
}

type integerIdSubscriptionResult struct {
	ID     int             `json:"subscription"`
	Result json.RawMessage `json:"result,omitempty"`
}

type tendermintSubscriptionResult struct {
	Query string `json:"query"`
}

// A value of this type can a JSON-RPC request, notification, successful response or
// error response. Which one it is depends on the fields.
type JsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Error   *JsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type tendermintSubscribeReply struct {
	Query string `json:"query"`
}

func (msg *JsonrpcMessage) isStarkNetPathfinderNotification() bool {
	return msg.ID == nil && msg.Method != "" && msg.Result != nil
}

func (msg *JsonrpcMessage) isEthereumNotification() bool {
	return msg.ID == nil && msg.Method != "" && msg.Params != nil
}

func (msg *JsonrpcMessage) isTendermintNotification() bool {
	var result tendermintSubscribeReply
	err := json.Unmarshal(msg.Result, &result)
	if err == nil && result.Query != "" {
		return true
	}
	return false
}

func (msg *JsonrpcMessage) isCall() bool {
	return msg.hasValidID() && msg.Method != ""
}

func (msg *JsonrpcMessage) isResponse() bool {
	return msg.hasValidID() && msg.Method == "" && msg.Params == nil && (msg.Result != nil || msg.Error != nil)
}

func (msg *JsonrpcMessage) hasValidID() bool {
	return len(msg.ID) > 0 && msg.ID[0] != '{' && msg.ID[0] != '['
}

func (msg *JsonrpcMessage) isSubscribe() bool {
	return strings.HasSuffix(msg.Method, subscribeMethodSuffix)
}

func (msg *JsonrpcMessage) isUnsubscribe() bool {
	return strings.HasSuffix(msg.Method, unsubscribeMethodSuffix)
}

func (msg *JsonrpcMessage) namespace() string {
	elem := strings.SplitN(msg.Method, serviceMethodSeparator, 2)
	return elem[0]
}

func (msg *JsonrpcMessage) String() string {
	b, _ := json.Marshal(msg)
	return string(b)
}

func (msg *JsonrpcMessage) errorResponse(err error) *JsonrpcMessage {
	resp := errorMessage(err)
	resp.ID = msg.ID
	return resp
}

func (msg *JsonrpcMessage) response(result interface{}) *JsonrpcMessage {
	enc, err := json.Marshal(result)
	if err != nil {
		// TODO: wrap with 'internal server error'
		return msg.errorResponse(err)
	}
	return &JsonrpcMessage{Version: Vsn, ID: msg.ID, Result: enc}
}

func errorMessage(err error) *JsonrpcMessage {
	msg := &JsonrpcMessage{Version: Vsn, ID: null, Error: &JsonError{
		Code:    defaultErrorCode,
		Message: err.Error(),
	}}
	ec, ok := err.(Error)
	if ok {
		msg.Error.Code = ec.ErrorCode()
	}
	de, ok := err.(DataError)
	if ok {
		msg.Error.Data = de.ErrorData()
	}
	return msg
}

type JsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Name    interface{} `json:"name,omitempty"`
	Cause   interface{} `json:"cause,omitempty"`
}

func (err *JsonError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("json-rpc error %d", err.Code)
	}
	return err.Message
}

func (err *JsonError) ErrorCode() int {
	return err.Code
}

func (err *JsonError) ErrorData() interface{} {
	return err.Data
}

func (err *JsonError) ToMap() map[string]interface{} {
	if err == nil {
		return nil
	}

	return map[string]interface{}{
		"code":    err.Code,
		"message": err.Message,
		"data":    err.Data,
		"name":    err.Name,
		"cause":   err.Cause,
	}
}

// Conn is a subset of the methods of net.Conn which are sufficient for ServerCodec.
type Conn interface {
	io.ReadWriteCloser
	SetWriteDeadline(time.Time) error
}

type deadlineCloser interface {
	io.Closer
	SetWriteDeadline(time.Time) error
}

// ConnRemoteAddr wraps the RemoteAddr operation, which returns a description
// of the peer address of a connection. If a Conn also implements ConnRemoteAddr, this
// description is used in log messages.
type ConnRemoteAddr interface {
	RemoteAddr() string
}

// jsonCodec reads and writes JSON-RPC messages to the underlying connection. It also has
// support for parsing arguments and serializing (result) objects.
type jsonCodec struct {
	remote  string
	closer  sync.Once                 // close closed channel once
	closeCh chan interface{}          // closed on Close
	decode  func(v interface{}) error // decoder to allow multiple transports
	encMu   utils.LavaMutex           // guards the encoder
	encode  func(v interface{}) error // encoder to allow multiple transports
	conn    deadlineCloser
}

// NewFuncCodec creates a codec which uses the given functions to read and write. If conn
// implements ConnRemoteAddr, log messages will use it to include the remote address of
// the connection.
func NewFuncCodec(conn deadlineCloser, encode, decode func(v interface{}) error) ServerCodec {
	codec := &jsonCodec{
		closeCh: make(chan interface{}),
		encode:  encode,
		decode:  decode,
		conn:    conn,
	}
	if ra, ok := conn.(ConnRemoteAddr); ok {
		codec.remote = ra.RemoteAddr()
	}
	return codec
}

// NewCodec creates a codec on the given connection. If conn implements ConnRemoteAddr, log
// messages will use it to include the remote address of the connection.
func NewCodec(conn Conn) ServerCodec {
	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)
	dec.UseNumber()
	return NewFuncCodec(conn, enc.Encode, dec.Decode)
}

func (c *jsonCodec) peerInfo() PeerInfo {
	// This returns "ipc" because all other built-in transports have a separate codec type.
	return PeerInfo{Transport: "ipc", RemoteAddr: c.remote}
}

func (c *jsonCodec) remoteAddr() string {
	return c.remote
}

func (c *jsonCodec) readBatch() (messages []*JsonrpcMessage, batch bool, err error) {
	// Decode the next JSON object in the input stream.
	// This verifies basic syntax, etc.
	var rawmsg json.RawMessage
	if err := c.decode(&rawmsg); err != nil {
		return nil, false, err
	}
	messages, batch = parseMessage(rawmsg)
	for i, msg := range messages {
		if msg == nil {
			// Message is JSON 'null'. Replace with zero value so it
			// will be treated like any other invalid message.
			messages[i] = new(JsonrpcMessage)
		}
	}
	return messages, batch, nil
}

func (c *jsonCodec) writeJSON(ctx context.Context, v interface{}) error {
	c.encMu.Lock()
	defer c.encMu.Unlock()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultWriteTimeout)
	}
	c.conn.SetWriteDeadline(deadline)
	return c.encode(v)
}

func (c *jsonCodec) close() {
	c.closer.Do(func() {
		close(c.closeCh)
		c.conn.Close()
	})
}

// Closed returns a channel which will be closed when Close is called
func (c *jsonCodec) closed() <-chan interface{} {
	return c.closeCh
}

// parseMessage parses raw bytes as a (batch of) JSON-RPC message(s). There are no error
// checks in this function because the raw message has already been syntax-checked when it
// is called. Any non-JSON-RPC messages in the input return the zero value of
// jsonrpcMessage.
func parseMessage(raw json.RawMessage) ([]*JsonrpcMessage, bool) {
	if !isBatch(raw) {
		msgs := []*JsonrpcMessage{{}}
		json.Unmarshal(raw, &msgs[0])
		return msgs, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.Token() // skip '['
	var msgs []*JsonrpcMessage
	for dec.More() {
		msgs = append(msgs, new(JsonrpcMessage))
		dec.Decode(&msgs[len(msgs)-1])
	}
	return msgs, true
}

// isBatch returns true when the first non-whitespace characters is '['
func isBatch(raw json.RawMessage) bool {
	for _, c := range raw {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}

// parsePositionalArguments tries to parse the given args to an array of values with the
// given types. It returns the parsed values or an error when the args could not be
// parsed. Missing optional arguments are returned as reflect.Zero values.
func parsePositionalArguments(rawArgs json.RawMessage, types []reflect.Type) ([]reflect.Value, error) {
	dec := json.NewDecoder(bytes.NewReader(rawArgs))
	var args []reflect.Value
	tok, err := dec.Token()
	switch {
	case err == io.EOF || tok == nil && err == nil:
		// "params" is optional and may be empty. Also allow "params":null even though it's
		// not in the spec because our own client used to send it.
	case err != nil:
		return nil, err
	case tok == json.Delim('['):
		// Read argument array.
		if args, err = parseArgumentArray(dec, types); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("non-array args")
	}
	// Set any missing args to nil.
	for i := len(args); i < len(types); i++ {
		if types[i].Kind() != reflect.Ptr {
			return nil, fmt.Errorf("missing value for required argument %d", i)
		}
		args = append(args, reflect.Zero(types[i]))
	}
	return args, nil
}

func parseArgumentArray(dec *json.Decoder, types []reflect.Type) ([]reflect.Value, error) {
	args := make([]reflect.Value, 0, len(types))
	for i := 0; dec.More(); i++ {
		if i >= len(types) {
			return args, fmt.Errorf("too many arguments, want at most %d", len(types))
		}
		argval := reflect.New(types[i])
		if err := dec.Decode(argval.Interface()); err != nil {
			return args, fmt.Errorf("invalid argument %d: %v", i, err)
		}
		if argval.IsNil() && types[i].Kind() != reflect.Ptr {
			return args, fmt.Errorf("missing value for required argument %d", i)
		}
		args = append(args, argval.Elem())
	}
	// Read end of args array.
	_, err := dec.Token()
	return args, err
}

// parseSubscriptionName extracts the subscription name from an encoded argument array.
func parseSubscriptionName(rawArgs json.RawMessage) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(rawArgs))
	if tok, _ := dec.Token(); tok != json.Delim('[') {
		return "", errors.New("non-array args")
	}
	v, _ := dec.Token()
	method, ok := v.(string)
	if !ok {
		return "", errors.New("expected subscription name as first argument")
	}
	return method, nil
}
