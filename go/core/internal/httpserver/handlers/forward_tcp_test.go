package handlers

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

// TestRawCodec verifies the rawCodec marshals and unmarshals []byte transparently.
func TestRawCodec(t *testing.T) {
	c := rawCodec{}
	if c.Name() != "proto" {
		t.Fatalf("Name() = %q, want %q", c.Name(), "proto")
	}

	original := []byte("hello world")
	encoded, err := c.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal([]byte): %v", err)
	}
	if !bytes.Equal(encoded, original) {
		t.Fatalf("Marshal round-trip mismatch: got %v, want %v", encoded, original)
	}

	ptrInput := &original
	encoded2, err := c.Marshal(ptrInput)
	if err != nil {
		t.Fatalf("Marshal(*[]byte): %v", err)
	}
	if !bytes.Equal(encoded2, original) {
		t.Fatalf("Marshal(*[]byte) mismatch: got %v, want %v", encoded2, original)
	}

	var decoded []byte
	if err := c.Unmarshal(original, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !bytes.Equal(decoded, original) {
		t.Fatalf("Unmarshal mismatch: got %v, want %v", decoded, original)
	}
}

// TestRawCodecUnsupportedTypes verifies that Marshal/Unmarshal return errors for unexpected types.
func TestRawCodecUnsupportedTypes(t *testing.T) {
	c := rawCodec{}
	if _, err := c.Marshal(42); err == nil {
		t.Fatal("expected error marshaling int")
	}
	var s string
	if err := c.Unmarshal([]byte("data"), &s); err == nil {
		t.Fatal("expected error unmarshaling into *string")
	}
}

// TestEncodeTcpForwardInitFrame verifies the wire encoding of the init frame.
func TestEncodeTcpForwardInitFrame(t *testing.T) {
	frame := encodeTcpForwardInitFrame("sb-001", "ssh-proxy:sb-001", "tok-abc")

	// Frame must be a TcpForwardFrame with field 1 (init) as an embedded message.
	fieldNum, wtyp, n := protowire.ConsumeTag(frame)
	if fieldNum != 1 || wtyp != protowire.BytesType {
		t.Fatalf("outer tag: field=%d type=%v", fieldNum, wtyp)
	}
	inner, m := protowire.ConsumeBytes(frame[n:])
	if m < 0 {
		t.Fatal("failed to decode inner TcpForwardInit bytes")
	}

	// Parse the inner TcpForwardInit.
	fields := map[protowire.Number]string{}
	b := inner
	for len(b) > 0 {
		fn, wt, tn := protowire.ConsumeTag(b)
		if tn < 0 {
			break
		}
		b = b[tn:]
		if wt == protowire.BytesType {
			val, bn := protowire.ConsumeBytes(b)
			if bn < 0 {
				break
			}
			b = b[bn:]
			fields[fn] = string(val)
		} else {
			bm := protowire.ConsumeFieldValue(fn, wt, b)
			if bm < 0 {
				break
			}
			b = b[bm:]
		}
	}

	if fields[1] != "sb-001" {
		t.Errorf("sandbox_id: got %q, want %q", fields[1], "sb-001")
	}
	if fields[4] != "ssh-proxy:sb-001" {
		t.Errorf("service_id: got %q, want %q", fields[4], "ssh-proxy:sb-001")
	}
	if fields[7] != "tok-abc" {
		t.Errorf("authorization_token: got %q, want %q", fields[7], "tok-abc")
	}
}

// TestEncodeTcpDataFrame verifies the wire encoding of a data frame.
func TestEncodeTcpDataFrame(t *testing.T) {
	payload := []byte("rawbytes")
	frame := encodeTcpDataFrame(payload)

	fieldNum, wtyp, n := protowire.ConsumeTag(frame)
	if fieldNum != 2 || wtyp != protowire.BytesType {
		t.Fatalf("data frame tag: field=%d type=%v", fieldNum, wtyp)
	}
	data, m := protowire.ConsumeBytes(frame[n:])
	if m < 0 {
		t.Fatal("failed to decode data bytes")
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("data mismatch: got %v, want %v", data, payload)
	}
}

// TestDecodeTcpDataFrame verifies round-trip encode/decode of a data frame.
func TestDecodeTcpDataFrame(t *testing.T) {
	want := []byte("hello grpc")
	frame := encodeTcpDataFrame(want)

	got, ok := decodeTcpDataFrame(frame)
	if !ok {
		t.Fatal("decodeTcpDataFrame: not ok")
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("decoded %v, want %v", got, want)
	}
}

// TestDecodeTcpDataFrame_NonDataFrame verifies that an init frame is not decoded as data.
func TestDecodeTcpDataFrame_NonDataFrame(t *testing.T) {
	initFrame := encodeTcpForwardInitFrame("sb", "svc", "tok")
	if data, ok := decodeTcpDataFrame(initFrame); ok {
		t.Fatalf("expected ok=false for init frame, got data=%v", data)
	}
}

// TestDecodeTcpDataFrame_Empty verifies that an empty byte slice returns not-ok.
func TestDecodeTcpDataFrame_Empty(t *testing.T) {
	if _, ok := decodeTcpDataFrame([]byte{}); ok {
		t.Fatal("expected ok=false for empty input")
	}
}
