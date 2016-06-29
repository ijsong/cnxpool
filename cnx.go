package cnxpool

import (
	"net"
	"time"
)

const readDeadlineInMillis = 5 * time.Millisecond
var zeroTime = time.Time{}

type cnx struct {
	net.Conn
	cp *CnxPool
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

func (c *cnx) watch(onSuccess func(conn *cnx), onFailure func(conn *cnx)) {
	go func(c *cnx) {
		buf := make([]byte, 1)
		c.SetReadDeadline(time.Now().Add(readDeadlineInMillis))
		defer c.SetReadDeadline(zeroTime)
		_, err := c.Read(buf)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				onSuccess(c)
			} else {
				onFailure(c)
			}
		}
	}(c)

}

