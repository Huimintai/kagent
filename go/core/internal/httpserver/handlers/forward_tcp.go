package handlers

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protowire"
)

// forwardTcpFullMethod is the gRPC full method name for the ForwardTcp bidirectional stream.
// The openshell server implements this natively (same as the openclaw CLI uses).
const forwardTcpFullMethod = "/openshell.v1.OpenShell/ForwardTcp"

// rawCodec bypasses protobuf serialization so that SendMsg/RecvMsg operate
// directly on []byte.  The codec name must be "proto" to replace the default
// codec registered by grpc-go.
type rawCodec struct{}

func (rawCodec) Marshal(v any) ([]byte, error) {
	switch b := v.(type) {
	case []byte:
		return b, nil
	case *[]byte:
		return *b, nil
	}
	return nil, fmt.Errorf("rawCodec: unsupported marshal type %T", v)
}

func (rawCodec) Unmarshal(data []byte, v any) error {
	if bp, ok := v.(*[]byte); ok {
		*bp = data
		return nil
	}
	return fmt.Errorf("rawCodec: unsupported unmarshal type %T", v)
}

func (rawCodec) Name() string { return "proto" }

// encodeTcpForwardInitFrame encodes the first TcpForwardFrame that carries a
// TcpForwardInit.  Field layout (proto3 wire format):
//
//	TcpForwardFrame.payload oneof → field 1 (init) = embedded TcpForwardInit message
//	  TcpForwardInit.sandbox_id   → field 1 (string)
//	  TcpForwardInit.service_id   → field 4 (string)
//	  TcpForwardInit.ssh target   → field 5 (embedded SshRelayTarget = empty message)
//	  TcpForwardInit.auth_token   → field 7 (string)
func encodeTcpForwardInitFrame(sandboxID, serviceID, token string) []byte {
	// Encode the inner TcpForwardInit message.
	var inner []byte
	inner = protowire.AppendTag(inner, 1, protowire.BytesType)
	inner = protowire.AppendString(inner, sandboxID)

	inner = protowire.AppendTag(inner, 4, protowire.BytesType)
	inner = protowire.AppendString(inner, serviceID)

	// SshRelayTarget is an empty message – encode as zero-length bytes.
	inner = protowire.AppendTag(inner, 5, protowire.BytesType)
	inner = protowire.AppendBytes(inner, []byte{})

	inner = protowire.AppendTag(inner, 7, protowire.BytesType)
	inner = protowire.AppendString(inner, token)

	// Wrap in TcpForwardFrame: field 1 = init (embedded message).
	var frame []byte
	frame = protowire.AppendTag(frame, 1, protowire.BytesType)
	frame = protowire.AppendBytes(frame, inner)
	return frame
}

// encodeTcpDataFrame encodes a TcpForwardFrame carrying raw bytes in field 2 (data).
func encodeTcpDataFrame(data []byte) []byte {
	var frame []byte
	frame = protowire.AppendTag(frame, 2, protowire.BytesType)
	frame = protowire.AppendBytes(frame, data)
	return frame
}

// decodeTcpDataFrame extracts the raw bytes from TcpForwardFrame.data (field 2).
// Returns (data, true) if the frame contains a data payload, (nil, false) otherwise.
func decodeTcpDataFrame(b []byte) ([]byte, bool) {
	for len(b) > 0 {
		fieldNum, wtyp, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, false
		}
		b = b[n:]
		if fieldNum == 2 && wtyp == protowire.BytesType {
			data, m := protowire.ConsumeBytes(b)
			if m < 0 {
				return nil, false
			}
			return data, true
		}
		// Skip unknown fields.
		m := protowire.ConsumeFieldValue(fieldNum, wtyp, b)
		if m < 0 {
			return nil, false
		}
		b = b[m:]
	}
	return nil, false
}

// forwardTcpConn wraps a gRPC bidirectional stream and exposes it as a net.Conn.
// The grpcConn passed in must NOT be closed externally; Close() takes ownership.
type forwardTcpConn struct {
	stream   grpc.ClientStream
	grpcConn *grpc.ClientConn

	mu  sync.Mutex
	buf []byte // leftover bytes from the last received frame
}

// openForwardTcpStream dials the ForwardTcp RPC on an existing grpc.ClientConn,
// sends the init frame, and returns a net.Conn wrapping the stream.
// On error the caller is responsible for closing grpcConn.
func openForwardTcpStream(ctx context.Context, cc *grpc.ClientConn, sandboxID, token string) (*forwardTcpConn, error) {
	desc := &grpc.StreamDesc{ServerStreams: true, ClientStreams: true}
	stream, err := cc.NewStream(ctx, desc, forwardTcpFullMethod, grpc.ForceCodec(rawCodec{}))
	if err != nil {
		return nil, fmt.Errorf("ForwardTcp NewStream: %w", err)
	}

	serviceID := "ssh-proxy:" + sandboxID
	initFrame := encodeTcpForwardInitFrame(sandboxID, serviceID, token)
	if err := stream.SendMsg(initFrame); err != nil {
		return nil, fmt.Errorf("ForwardTcp send init: %w", err)
	}

	return &forwardTcpConn{stream: stream, grpcConn: cc}, nil
}

func (c *forwardTcpConn) Read(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Drain leftover buffer first.
	if len(c.buf) > 0 {
		n := copy(b, c.buf)
		c.buf = c.buf[n:]
		return n, nil
	}

	// Receive the next frame from the stream.
	for {
		var raw []byte
		if err := c.stream.RecvMsg(&raw); err != nil {
			return 0, err
		}
		data, ok := decodeTcpDataFrame(raw)
		if !ok || len(data) == 0 {
			// Non-data frame or empty – skip and read the next one.
			continue
		}
		n := copy(b, data)
		if n < len(data) {
			// Save excess bytes for the next Read call.
			leftover := make([]byte, len(data)-n)
			copy(leftover, data[n:])
			c.buf = leftover
		}
		return n, nil
	}
}

func (c *forwardTcpConn) Write(b []byte) (int, error) {
	frame := encodeTcpDataFrame(b)
	if err := c.stream.SendMsg(frame); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *forwardTcpConn) Close() error {
	_ = c.stream.CloseSend()
	return c.grpcConn.Close()
}

// Stub net.Conn methods – deadlines are managed at the gRPC/context level.

func (c *forwardTcpConn) LocalAddr() net.Addr              { return stubAddr{} }
func (c *forwardTcpConn) RemoteAddr() net.Addr             { return stubAddr{} }
func (c *forwardTcpConn) SetDeadline(_ time.Time) error    { return nil }
func (c *forwardTcpConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *forwardTcpConn) SetWriteDeadline(_ time.Time) error { return nil }

type stubAddr struct{}

func (stubAddr) Network() string { return "grpc" }
func (stubAddr) String() string  { return "openshell-forward-tcp" }
