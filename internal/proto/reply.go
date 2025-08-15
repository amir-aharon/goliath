package proto

import (
	"fmt"
	"io"
)

func OK(w io.Writer) error {
	_, err := fmt.Fprint(w, "+OK\r\n")
	return err
}

func Err(w io.Writer, msg string) error {
	_, err := fmt.Fprintf(w, "-ERR %s\r\n", msg)
	return err
}

func PONG(w io.Writer) error {
	_, err := fmt.Fprint(w, "+PONG\r\n")
	return err
}

func Line(w io.Writer, text string) error {
	_, err := fmt.Fprintf(w, "%s\r\n", text)
	return err
}

func Bulk(w io.Writer, s string) error {
	_, err := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(s), s)
	return err
}

func Int(w io.Writer, n int64) error {
	_, err := fmt.Fprintf(w, "%d\r\n", n)
	return err
}
