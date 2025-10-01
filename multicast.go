package multicast

import (
	"errors"
	"fmt"
	"net"
	"runtime"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// UDPConn is a UDP connection configured for multicast communication.
// It embeds net.UDPConn and provides convenience methods for multicast
// reads/writes.
type UDPConn struct {
	net.UDPConn

	// network is "udp4" or "udp6".
	network string
	// ifaces lists interfaces joined to the multicast group.
	ifaces []net.Interface
	// ipv4conn is non-nil when network == "udp4".
	ipv4conn *ipv4.PacketConn
	// ipv6conn is non-nil when network == "udp6".
	ipv6conn *ipv6.PacketConn
}

// ListenMulticastUDPIfaces listens for multicast on the provided address and
// joins multicast groups on the given network interfaces.
//
// It accepts "udp4" or "udp6" for the network argument.
// If ifaces is nil, it will use all multicast-capable interfaces.
// The addr argument specifies the socket to bind to.
// It returns a *UDPConn ready for multicast reads/writes.
func ListenMulticastUDPIfaces(network string, ifaces []net.Interface, addr *net.UDPAddr) (*UDPConn, error) {
	if addr == nil {
		return nil, errors.New("multicast: addr cannot be nil")
	}

	switch network {
	case "udp4", "udp6":
	default:
		return nil, fmt.Errorf("network must be either 'udp4' or 'udp6': %s", network)
	}

	udpConn, err := net.ListenUDP(network, addr)
	if err != nil {
		return nil, err
	}

	if ifaces == nil {
		ifaces, err = multicastInterfaces()
		if err != nil {
			return nil, fmt.Errorf("multicast: failed to get multicast interfaces: %w", err)
		}
	}

	var v4PkConn *ipv4.PacketConn
	var v6PkConn *ipv6.PacketConn
	switch network {
	case "udp4":
		v4PkConn = ipv4.NewPacketConn(udpConn)
	case "udp6":
		v6PkConn = ipv6.NewPacketConn(udpConn)
	}

	conn := &UDPConn{
		UDPConn:  *udpConn,
		network:  network,
		ifaces:   ifaces,
		ipv4conn: v4PkConn,
		ipv6conn: v6PkConn,
	}

	ok, err := conn.joinIfaces(ifaces, addr)
	if !ok && err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

// Close closes the underlying connection.
func (c *UDPConn) Close() error {
	var err error
	if c.ipv4conn != nil {
		err = errors.Join(err, c.ipv4conn.Close())
		c.ipv4conn = nil
	}
	if c.ipv6conn != nil {
		err = errors.Join(err, c.ipv6conn.Close())
		c.ipv6conn = nil
	}
	err = errors.Join(err, c.UDPConn.Close())
	return err
}

// JoinMulticastGroup joins the multicast group gaddr on iface.
// This allows adding additional interfaces to the multicast group dynamically.
//
// Example:
//
//	addr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
//	conn, err := multicast.ListenMulticastUDPIfaces("udp4", nil, addr)
//	gaddr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
//	err := conn.JoinMulticastGroup(iface, gaddr)
func (c *UDPConn) JoinMulticastGroup(iface *net.Interface, gaddr *net.UDPAddr) error {
	if iface == nil {
		return errors.New("multicast: interface cannot be nil")
	}
	if gaddr == nil {
		return errors.New("multicast: group address cannot be nil")
	}

	switch c.network {
	case "udp4":
		err := c.ipv4conn.JoinGroup(iface, gaddr)
		if err != nil {
			return err
		}
	case "udp6":
		err := c.ipv6conn.JoinGroup(iface, gaddr)
		if err != nil {
			return err
		}
	default:
		panic("unreachable")
	}

	c.ifaces = append(c.ifaces, *iface)
	return nil
}

// SetMulticastTTL sets the multicast TTL (IPv4) or hop limit (IPv6) used for
// outbound multicast packets.
func (c *UDPConn) SetMulticastTTL(ttl int) error {
	switch c.network {
	case "udp4":
		return c.ipv4conn.SetMulticastTTL(ttl)
	case "udp6":
		return c.ipv6conn.SetMulticastHopLimit(ttl)
	default:
		panic("unreachable")
	}
}

// SetMulticastHopLimit is an alias to SetMulticastTTL for API symmetry for IPv6.
func (c *UDPConn) SetMulticastHopLimit(hoplim int) error {
	return c.SetMulticastTTL(hoplim)
}

// SetMulticastLoopback sets whether multicast packets sent from this socket
// should be looped back to the local sockets.
func (c *UDPConn) SetMulticastLoopback(on bool) error {
	switch c.network {
	case "udp4":
		return c.ipv4conn.SetMulticastLoopback(on)
	case "udp6":
		return c.ipv6conn.SetMulticastLoopback(on)
	default:
		panic("unreachable")
	}
}

// WriteToMulticast sends buf to the multicast address addr using all joined
// interfaces. Any errors encountered during transmission on each interface
// are aggregated and returned as a joined error.
func (c *UDPConn) WriteToMulticast(buf []byte, addr *net.UDPAddr) error {
	if addr == nil {
		return errors.New("multicast: address cannot be nil")
	}
	if len(buf) == 0 {
		return nil
	}

	var errs []error

	switch c.network {
	case "udp4":
		var wcm ipv4.ControlMessage
		for ifi := range c.ifaces {
			switch runtime.GOOS {
			case "darwin", "ios", "linux":
				wcm.IfIndex = c.ifaces[ifi].Index
			default:
				if err := c.ipv4conn.SetMulticastInterface(&c.ifaces[ifi]); err != nil {
					errs = append(errs, err)
				}
			}
			if _, err := c.ipv4conn.WriteTo(buf, &wcm, addr); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	case "udp6":
		var wcm ipv6.ControlMessage
		for ifi := range c.ifaces {
			switch runtime.GOOS {
			case "darwin", "ios", "linux":
				wcm.IfIndex = c.ifaces[ifi].Index
			default:
				if err := c.ipv6conn.SetMulticastInterface(&c.ifaces[ifi]); err != nil {
					errs = append(errs, err)
				}
			}
			if _, err := c.ipv6conn.WriteTo(buf, &wcm, addr); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	default:
		panic("unreachable")
	}
}

func (c *UDPConn) joinIfaces(ifaces []net.Interface, gaddr *net.UDPAddr) (ok bool, err error) {
	var errs error
	var fails int

	for _, iface := range ifaces {
		if err := c.JoinMulticastGroup(&iface, gaddr); err != nil {
			fails++
			errs = errors.Join(errs, err)
		}
	}
	if fails == len(ifaces) {
		return false, fmt.Errorf("udp: failed to join any interface: %w", errs)
	}

	if errs != nil && fails < len(ifaces) {
		return true, fmt.Errorf("multicast: failed to join %d/%d interfaces: %w", fails, len(ifaces), errs)
	}
	return true, nil
}

func multicastInterfaces() ([]net.Interface, error) {
	var mifaces []net.Interface
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, ifi := range ifaces {
		if (ifi.Flags&net.FlagUp) != 0 && (ifi.Flags&net.FlagMulticast) != 0 {
			mifaces = append(mifaces, ifi)
		}
	}

	return mifaces, nil
}

// Network returns the network type ("udp4" or "udp6") of the connection.
func (c *UDPConn) Network() string {
	if c == nil {
		return ""
	}
	return c.network
}

// Interfaces returns a copy of the interfaces that have joined the multicast group.
func (c *UDPConn) Interfaces() []net.Interface {
	if c == nil {
		return nil
	}
	// Return a copy to prevent external modification
	ifaces := make([]net.Interface, len(c.ifaces))
	copy(ifaces, c.ifaces)
	return ifaces
}
