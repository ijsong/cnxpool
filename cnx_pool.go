// Package cnxpool provides connection pool which handles connection failure silently.
package cnxpool

import (
	"net"
	"time"
	"errors"
)

// Connection Pool type.
type CnxPool struct {
	rq chan *cnx
	connect func() (net.Conn, error)
	capacity int
	watchdogTicker *time.Ticker
}

const (
	defaultCnxPoolCapacity = 32 // default the number of connections in pool
	watchdogTick = 10 * time.Millisecond // watchdog interval
)

// NewCnxPool creates connection pool.
// First argument 'connect' is a function which makes connection.
// See https://golang.org/pkg/net/#Dial.
// Argument 'capacity' is the number of connections in pool.
// Initially, created connection pool has 'capacity' connections.
// When connection is not enough in pool, it generates new connection by using function 'connect'.
func NewCnxPool(connect func() (net.Conn, error), capacity int) (*CnxPool, error) {
	if capacity <= 0 {
		return nil, errors.New("Argument 'capacity' should be positive number.")
	}

	cp := &CnxPool{
		rq: make(chan *cnx, capacity),
		connect: connect,
		capacity: capacity,
	}

	for i := 0; i < capacity; i++ {
		conn, err := cp.connect()
		if err != nil {
			return nil, errors.New("CnxPool can't use connect function")
		}
		cp.rq <- newCnx(conn, cp)
	}

	cp.watchdog()

	return cp, nil
}

// Get returns connection which is compatible with net.Conn.
// See https://golang.org/pkg/net/#Conn.
func (cp *CnxPool) Get() (net.Conn, error) {
	return cp.get()
}

// Close destroys connection pool.
func (cp *CnxPool) Close() {
	cp.watchdogTicker.Stop()
	close(cp.rq)
	for conn := range cp.rq {
		conn.close()
	}
}

func (cp *CnxPool) get() (net.Conn, error) {
	if conn := cp.demote(); conn != nil {
		return conn, nil
	}

	conn, err := cp.connect()
	if err != nil {
		return nil, err
	}

	return newCnx(conn, cp), nil
}

func (cp *CnxPool) release(conn *cnx) error {
	onSuccess := func(conn *cnx) {
		conn.cp.promote(conn)
	}

	onFailure := func(conn *cnx) {
		conn.close()
	}

	conn.watch(onSuccess, onFailure)
	return nil
}

func (cp *CnxPool) watchdog() {
	onSuccess := func(conn *cnx) {
		conn.cp.promote(conn)
	}

	onFailure := func(conn *cnx) {
		conn.close()
	}

	cp.watchdogTicker = time.NewTicker(watchdogTick)

	go func(cp *CnxPool) {
		for {
			select {
			case <-cp.watchdogTicker.C:
				if conn := cp.demote(); conn != nil {
					conn.watch(onSuccess, onFailure)
				}
			}
		}
	}(cp)
}

func (cp *CnxPool) demote() *cnx {
	select {
	case conn := <-cp.rq:
		return conn
	default:
		return nil
	}
}

func (cp *CnxPool) promote(conn *cnx) error {
	select {
	case cp.rq <- conn:
		return nil
	default:
		return conn.close()
	}
}
