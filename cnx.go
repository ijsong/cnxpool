package cnxpool

import (
	"log"
	"net"
	"time"
	"sync"
)

var _ = log.Println
var ZeroTime = time.Time{}

const (
	ReadDeadlineInMillis = 5 * time.Millisecond
)

type cnx struct {
	net.Conn
	cp *CnxPool
	mut sync.Mutex
	watchCount int
}

func newCnx(conn net.Conn, cp *CnxPool) *cnx {
	return &cnx{
		Conn: conn,
		cp: cp,
	}
}

func (c *cnx) Close() error {
	return c.cp.release(c)
}

func (c *cnx) close() error {
	return c.Conn.Close()
}

func (c *cnx) watch() {
	go func(c *cnx) {
		buf := make([]byte, 1)
		c.SetReadDeadline(time.Now().Add(ReadDeadlineInMillis))
		defer c.SetReadDeadline(ZeroTime)
		_, err := c.Read(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				c.cp.promote(c)
				return
			}
			c.close()
		}
	}(c)

}

