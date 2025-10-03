# multicast

A small Go helper library for UDP multicast with multi-interface support.

:::note This library is not thoroughly tested. Please test it carefully in your
environment and verify it behaves as expected before using it in production. Bug
reports and pull requests are welcome. :::

This package provides a thin wrapper to simplify IPv4/IPv6 multicast
send/receive. It supports joining multicast groups on multiple network
interfaces, setting TTL/HopLimit, controlling multicast loopback, and sending a
packet from all joined interfaces.

## Features

- Supports `udp4` and `udp6`
- Automatically discovers multicast-capable interfaces
- Dynamically add interfaces to a multicast group
- Set multicast TTL (IPv4) / hop limit (IPv6)
- Control multicast loopback
- Send a packet on all joined interfaces (`WriteToMulticast`)

## Installation

```bash
go get github.com/oosawy/multicast
```

## Usage (quick example)

An example that sends an mDNS query and listens for responses is available in
`examples/mdns`.

Build and run the example (fish shell):

```fish
cd examples/mdns
go build -o mdns-example
./mdns-example my-host.local
```

Core usage example (excerpt):

```go
addr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
conn, err := multicast.ListenMulticastUDPIfaces("udp4", nil, addr)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

gaddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
if err := conn.JoinMulticastGroup(nil, gaddr); err != nil {
    log.Printf("warning: failed to join some interfaces: %v", err)
}

_ = conn.SetMulticastTTL(255)
_ = conn.SetMulticastLoopback(true)
_ = conn.ReuseAddrPort()

// Send
buf := []byte("...")
_ = conn.WriteToMulticast(buf, gaddr)

// Receive
b := make([]byte, 65536)
n, _, err := conn.ReadFrom(b)
if err != nil {
    log.Printf("read error: %v", err)
}
```

## API summary

- `ListenMulticastUDPIfaces(network string, ifaces []net.Interface, addr *net.UDPAddr) (*UDPConn, error)`
  - Creates a socket bound to `addr` and joins the multicast group on the
    provided interfaces. If `ifaces` is nil, the package will enumerate
    multicast-capable interfaces.
  - `network` must be either `"udp4"` or `"udp6"`.

- `(*UDPConn) JoinMulticastGroup(ifaces []net.Interface, gaddr *net.UDPAddr) error`
  - Join additional multicast group on the interfaces. If `ifaces` is nil, the
    connection's current interfaces are used.

- `(*UDPConn) SetMulticastTTL(ttl int) error`
  - Set IPv4 TTL or IPv6 hop limit for outgoing multicast packets.

- `(*UDPConn) SetMulticastHopLimit(hopLimit int) error`
  - Set IPv6 hop limit for outgoing multicast packets. This is an alias to
    `SetMulticastTTL` for API symmetry to IPv6.

- `(*UDPConn) SetMulticastLoopback(on bool) error`
  - Enable or disable loopback of multicast packets to local sockets.

- `(*UDPConn) ReuseAddrPort() error`
  - Try to set SO_REUSEADDR/PORT on the underlying socket (platform dependent).

- `(*UDPConn) WriteToMulticast(buf []byte, addr *net.UDPAddr) error`
  - Send `buf` to `addr` using all joined interfaces. Errors per-interface are
    aggregated and returned.

- `(*UDPConn) Interfaces() []net.Interface` / `(*UDPConn) Network() string`

## Notes

- Multicast and low-level socket behavior varies between OS. On some systems
  (notably macOS and some Linux distributions) additional permissions or sysctl
  settings may be required.
- Behavior of `ReuseAddrPort` is OS-dependent.
- When attempting to join multiple interfaces at `ListenMulticastUDPIfaces` or
  `JoinMulticastGroup`, if some (but not all) interfaces fail to join, the
  library logs a warning via the `slog` package describing the partial failure.
  The connection will still be returned if at least one interface succeeded.

## Acknowledgements

This library's multicast sending logic was partly inspired by
[grandcat/zeroconf](https://github.com/grandcat/zeroconf); thanks to its
maintainers and contributors for their work.

## License

This repository follows the terms in the `LICENSE` file.

## Contributing

Bug reports and pull requests are welcome. Please open an issue first for larger
changes.
