package cnxpool

import (
	"net"
	"time"
	"log"
)

type CnxPool struct {
	rq chan *cnx
	connect func() (net.Conn, error)
	capacity int
	watchdogTicker *time.Ticker
}

var WatchdogTick = 10 * time.Millisecond

func NewCnxPool(network, address string, capacity int) *CnxPool {
	cp := &CnxPool{
		rq: make(chan *cnx, capacity),
		connect: func() (net.Conn, error) { return net.Dial(network, address) },
		capacity: capacity,
	}

	for i := 0; i < capacity; i++ {
		conn, err := cp.connect()
		if err != nil {
			log.Println("failed to initialize CnxPool:", err)
			return nil
		}
		cp.rq <- newCnx(conn, cp)
	}

	cp.watchdog()

	return cp
}

func (cp *CnxPool) Get() (net.Conn, error) {
	return cp.get()
}

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

	cp.watchdogTicker = time.NewTicker(WatchdogTick)

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
