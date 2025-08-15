package server

import (
	"fmt"
	"log"
	"net"
)

type NewSession func(net.Conn) interface{ Run() }

// initialize session on new connection
type Server struct {
	Addr string
	Port int
	New  NewSession
}

func (srv *Server) Serve() error {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", srv.Addr, srv.Port))
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go srv.New(conn).Run()
	}
}
