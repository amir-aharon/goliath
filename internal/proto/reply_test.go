package proto_test

import (
	"bytes"
	"testing"

	"github.com/amir-aharon/goliath/internal/proto"
)

func TestOK(t *testing.T) {
	var buf bytes.Buffer
	err := proto.OK(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	want := "+OK\r\n"

	if got != want {
		t.Errorf("proto.OK() wrote %q, want %q", got, want)
	}
}

func TestPONG(t *testing.T) {
	var buf bytes.Buffer
	if err := proto.PONG(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "+PONG\r\n"; got != want {
		t.Errorf("proto.PONG() wrote %q, want %q", got, want)
	}
}

func TestLine(t *testing.T) {
	var buf bytes.Buffer
	if err := proto.Line(&buf, "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "hello\r\n"; got != want {
		t.Errorf("proto.Line() wrote %q, want %q", got, want)
	}
}

func TestErr(t *testing.T) {
	var buf bytes.Buffer
	if err := proto.Err(&buf, "key not found"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "-ERR key not found\r\n"; got != want {
		t.Errorf("proto.Err() wrote %q, want %q", got, want)
	}
}

func TestInt(t *testing.T) {
	tests := []struct {
		name string
		in   int64
		want string
	}{
		{"zero", 0, "0\r\n"},
		{"positive", 42, "42\r\n"},
		{"negative", -2, "-2\r\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := proto.Int(&buf, tc.in); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("proto.Int(%d) wrote %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestBulkASCII(t *testing.T) {
	var buf bytes.Buffer
	if err := proto.Bulk(&buf, "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "$5\r\nhello\r\n"; got != want {
		t.Errorf("proto.Bulk() wrote %q, want %q", got, want)
	}
}

func TestBulkUTF8(t *testing.T) {
	var buf bytes.Buffer
	s := "שלום" // 4 runes, 8 bytes in UTF-8
	if err := proto.Bulk(&buf, s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := buf.String(), "$8\r\nשלום\r\n"; got != want {
		t.Errorf("proto.Bulk() wrote %q, want %q", got, want)
	}
}

