package main

import (
	"log"
	"net"
	"os"
	"strconv"

	"github.com/amir-aharon/goliath/internal/command"
	"github.com/amir-aharon/goliath/internal/server"
	"github.com/amir-aharon/goliath/internal/session"
	"github.com/amir-aharon/goliath/internal/store"
)

func main() {
	port := 6379
	if p, err := strconv.Atoi(os.Getenv("PORT")); err == nil && p > 0 {
		port = p
	}

	d := command.NewDispatcher()
	command.RegisterBuiltins(d)

	kv := store.NewMemory()
	command.RegisterKV(d, kv)
	command.RegisterTTL(d, kv)

	srv := server.Server{
		Addr: "0.0.0.0",
		Port: port,
		New: func(c net.Conn) interface{ Run() } {
			return session.New(c, d)
		},
	}
	log.Fatal(srv.Serve())
}
