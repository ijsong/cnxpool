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
	select {
	case conn := <-cp.rq:
		return conn, nil
	default:
		conn, err := cp.connect()
		if err != nil {
			return nil, err
		}
		return newCnx(conn, cp), nil
	}
}

func (cp *CnxPool) release(conn *cnx) error {
	select {
	case cp.rq <- conn:
		return nil
	default:
		return conn.close()
	}
}

func (cp *CnxPool) watchdog() {
	cp.watchdogTicker = time.NewTicker(WatchdogTick)
	go func(cp *CnxPool) {
		for {
			select {
			case <-cp.watchdogTicker.C:
				cp.demote()
			}
		}
	}(cp)
}

func (cp *CnxPool) demote() {
	select {
	case conn := <-cp.rq:
		conn.watch()
	}
}

func (cp *CnxPool) promote(conn *cnx) {
	select {
	case cp.rq <- conn:
	default:
		conn.close()
	}
}
