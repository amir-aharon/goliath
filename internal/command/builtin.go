package command

import (
	"io"
	"strconv"
	"time"

	"github.com/amir-aharon/goliath/internal/proto"
	"github.com/amir-aharon/goliath/internal/store"
)

func RegisterBuiltins(r *Router) {
	r.Handle("PING", func(w io.Writer, _ []string) error {
		return proto.PONG(w)
	})
	r.Handle("ECHO", func(w io.Writer, args []string) error {
		if len(args) < 1 {
			return proto.Err(w, "wrong number of arguments for 'ECHO'")
		}
		return proto.Line(w, args[0])
	})
	r.Handle("QUIT", func(w io.Writer, _ []string) error {
		if err := proto.OK(w); err != nil {
			return err
		}
		return ErrQuit
	})
}

func RegisterKV(r *Router, kv store.KV) {
	r.Handle("GET", func(w io.Writer, args []string) error {
		if len(args) != 1 {
			return proto.Err(w, "wrong number of arguments for 'GET'")
		}
		if v, ok := kv.Get(args[0]); ok {
			return proto.Line(w, v)
		}
		return proto.Err(w, "key not found")
	})

	r.Handle("SET", func(w io.Writer, args []string) error {
		if len(args) != 2 {
			return proto.Err(w, "wrong number of arguments for 'SET'")
		}
		kv.Set(args[0], args[1])
		return proto.OK(w)
	})

	r.Handle("SETEX", func(w io.Writer, args []string) error {
		if len(args) != 3 {
			return proto.Err(w, "wrong number of arguments for 'SETEX'")
		}
		secs, err := strconv.Atoi(args[1])
		if err != nil || secs <= 0 {
			return proto.Err(w, "seconds must be a positive integer")
		}
		kv.SetEx(args[0], args[2], time.Duration(secs)*time.Second)
		return proto.OK(w)
	})

	r.Handle("DEL", func(w io.Writer, args []string) error {
		if len(args) != 1 {
			return proto.Err(w, "wrong number of arguments for 'DEL'")
		}
		if kv.Del(args[0]) {
			return proto.OK(w)
		}
		return proto.Err(w, "key not found")
	})
}

func RegisterTTL(r *Router, kv store.KV) {
	r.Handle("TTL", func(w io.Writer, args []string) error {
		if len(args) != 1 {
			return proto.Err(w, "wrong number of arguments for 'TTL'")
		}
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
}
