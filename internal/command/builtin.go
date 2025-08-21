package command

import (
	"io"
	"strconv"
	"time"

	"github.com/amir-aharon/goliath/internal/proto"
	"github.com/amir-aharon/goliath/internal/store"
)

func RegisterBuiltins(d *Dispatcher) {
	d.Register("PING", 0, 0, false, func(w io.Writer, _ []string) error {
		return proto.PONG(w)
	})
	d.Register("ECHO", 1, 1, false, func(w io.Writer, args []string) error {
		return proto.Line(w, args[0])
	})
	d.Register("QUIT", 0, 0, false, func(w io.Writer, _ []string) error {
		if err := proto.OK(w); err != nil {
			return err
		}
		return ErrQuit
	})
}

func RegisterKV(d *Dispatcher, kv store.KV) {
	d.Register("GET", 1, 1, false, func(w io.Writer, args []string) error {
		if v, ok := kv.Get(args[0]); ok {
			return proto.Line(w, v)
		}
		return proto.Err(w, "key not found")
	})

	d.Register("SET", 2, 2, true, func(w io.Writer, args []string) error {
		kv.Set(args[0], args[1])
		return proto.OK(w)
	})

	d.Register("SETEX", 3, 3, true, func(w io.Writer, args []string) error {
		secs, err := strconv.Atoi(args[1])
		if err != nil || secs <= 0 {
			return proto.Err(w, "seconds must be a positive integer")
		}
		kv.SetEx(args[0], args[2], time.Duration(secs)*time.Second)
		return proto.OK(w)
	})

	d.Register("DEL", 1, 1, true, func(w io.Writer, args []string) error {
		if kv.Del(args[0]) {
			return proto.OK(w)
		}
		return proto.Err(w, "key not found")
	})
}

func RegisterTTL(d *Dispatcher, kv store.KV) {
	d.Register("TTL", 1, 1, false, func(w io.Writer, args []string) error {
		secs, exists, hasExp := kv.TTL(args[0])
		switch {
		case !exists:
			return proto.Int(w, -2)
		case !hasExp:
			return proto.Int(w, -1)
		default:
			return proto.Int(w, int64(secs))
		}
	})

	d.Register("PERSIST", 1, 1, true, func(w io.Writer, args []string) error {
		hasExp := kv.Persist(args[0])
		if hasExp {
			return proto.Int(w, 1)
		}
		return proto.Int(w, 0)
	})
}
