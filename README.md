## About

PROXY protocol version 1 parser and `net.Listener` adapter.

## Install

```
go get github.com/poorlydefinedbehaviour/proxy-protocol-go
```

## Examples

### Parsing the protocol header

```go
header, err = proxyprotocol.ParseProtocolHeader([]byte("PROXY TCP4 255.255.255.255 255.255.255.254 65535 65534\r\n"))
if err != nil {
  return err
}
fmt.Printf("%+v\n", header)
// {
//   version: 1, 
//   inetProtocol: "TCP4",
//   srcPort: 65535,
//   destPort: 65534,
//   src: [0 0 0 0 0 0 0 0 0 0 255 255 255 255 255 255],
//   dest:[0 0 0 0 0 0 0 0 0 0 255 255 255 255 255 254],
// }
```

### Writing the header

```go
if err := proxyprotocol.WriteHeader(header, conn); err != nil {
  return err
}
```

### Wrapping a net.Listener

```go
ln, err := net.Listen("tcp", "localhost:9879")
if err != nil {
  return err
}

listener := proxyprotocol.ListenerAdapter{
  Listener: ln,
  // If the listener does not receive the proxy protocol header from a connection
  // after `ProxyProtocolHeaderReadTimeout`, an error is returned by `Accept`.
  ProxyProtocolHeaderReadTimeout: 5 * time.Second
}

conn, err := listener.Accept()
if err != nil {
  return err
}
defer conn.Close()
```

### Creating a http server that expects the PROXY protocol header before http requests

```go
ln, err := net.Listen("tcp", "localhost:9879")
if err != nil {
  return err
}

listener := proxyprotocol.ListenerAdapter{
  Listener: ln,
  // If the listener does not receive the proxy protocol header from a connection
  // after `ProxyProtocolHeaderReadTimeout`, an error is returned by `Accept`.
  ProxyProtocolHeaderReadTimeout: 5 * time.Second
}

if err := http.Serve(&listener, handler); err != nil {
  panic(err)
}
```

## References

https://www.haproxy.org/download/2.3/doc/proxy-protocol.txt