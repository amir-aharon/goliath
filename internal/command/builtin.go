package command

import (
	"io"
	"strconv"
	"time"

	"github.com/amir-aharon/goliath/internal/proto"
	"github.com/amir-aharon/goliath/internal/store"
)

func RegisterBuiltins(r *Router) {
	r.Handle("PING", 0, 0, func(w io.Writer, _ []string) error {
		return proto.PONG(w)
	})
	r.Handle("ECHO", 1, 1, func(w io.Writer, args []string) error {
		return proto.Line(w, args[0])
	})
	r.Handle("QUIT", 0, 0, func(w io.Writer, _ []string) error {
		if err := proto.OK(w); err != nil {
			return err
		}
		return ErrQuit
	})
}

func RegisterKV(r *Router, kv store.KV) {
	r.Handle("GET", 1, 1, func(w io.Writer, args []string) error {
		if v, ok := kv.Get(args[0]); ok {
			return proto.Line(w, v)
		}
		return proto.Err(w, "key not found")
	})

	r.Handle("SET", 2, 2, func(w io.Writer, args []string) error {
		kv.Set(args[0], args[1])
		return proto.OK(w)
	})

	r.Handle("SETEX", 3, 3, func(w io.Writer, args []string) error {
		secs, err := strconv.Atoi(args[1])
		if err != nil || secs <= 0 {
			return proto.Err(w, "seconds must be a positive integer")
		}
		kv.SetEx(args[0], args[2], time.Duration(secs)*time.Second)
		return proto.OK(w)
	})

	r.Handle("DEL", 1, 1, func(w io.Writer, args []string) error {
		if kv.Del(args[0]) {
			return proto.OK(w)
		}
		return proto.Err(w, "key not found")
	})
}

func RegisterTTL(r *Router, kv store.KV) {
	r.Handle("TTL", 1, 1, func(w io.Writer, args []string) error {
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

	r.Handle("PERSIST", 1, 1, func(w io.Writer, args []string) error {
		hasExp := kv.Persist(args[0])
		if hasExp {
			return proto.Int(w, 1)
		}
		return proto.Int(w, 0)
	})
}
