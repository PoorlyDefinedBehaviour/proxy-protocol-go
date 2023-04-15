## About

PROXY protocol version 1 parser.

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

## References

https://www.haproxy.org/download/2.3/doc/proxy-protocol.txt