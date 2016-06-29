package cnxpool

import (
	"net"
	"log"
	"testing"
)

const (
	network = "tcp"
	address = "127.0.0.1:8008"
)

var connector = func () (net.Conn, error) {
	return net.Dial(network, address)
}

func init() {
	done := make(chan bool, 1)
	go func() {
		ln, err := net.Listen(network, address)
		if err != nil {
			log.Fatalln(err)
		}
		defer ln.Close()
		done <- true
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Fatalln(err)
			}
			go func(conn net.Conn) {
				buf := make([]byte, 32)
				conn.Read(buf)
			}(conn)
		}
	}()
	<-done
}

func TestNewCnxPool(t *testing.T) {
	pool, err := NewCnxPool(connector, 10)
	if err != nil {
		t.Error("NewCnxPool error:", err)
	}
	defer pool.Close()
}

func TestGetAndClose(t *testing.T) {
	pool, err := NewCnxPool(connector, 10)
	if err != nil {
		t.Error("NewCnxPool error:", err)
	}
	defer pool.Close()

	conn, err := pool.Get()
	if err != nil {
		t.Error("Get error:", err)
	}

	if err := conn.Close(); err != nil {
		t.Error("Close error:", err)
	}
}

