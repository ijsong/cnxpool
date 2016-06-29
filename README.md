# cnxpool
[![Build Status](https://travis-ci.org/xsdb/cnxpool.svg?branch=master)](https://travis-ci.org/xsdb/cnxpool)

- Socket connection pool
- Handle closing connection silently

## Getting Started
```go
import (
  "net"
  "cnxpool"
)
```

```go
// Create connection pool
pool, err := cnxpool.NewCnxPool(func () (net.Conn, error) {
}, 32)
if err != nil {
  // handle error
}

// Get connection from connection pool
conn, err := pool.Get()
if err != nil {
  // handle error
}

// Close connection
err := conn.Close()
if err != nil {
  // handle error
}

// Destroy connection pool
pool.Close()
```
