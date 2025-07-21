package service

import (
	"context"
	"fmt"
	"github.com/mlctrez/bind"
	"github.com/nats-io/nats.go"
	"io"
	"log/slog"
	"net"
)

var _ bind.Startup = (*UDPBridge)(nil)
var _ bind.Shutdown = (*UDPBridge)(nil)

// UDPBridge bridges factorio udp_send and udp_rec traffic to and from NATS
type UDPBridge struct {
	slog.Logger
	Context      context.Context
	Nats         *Nats
	incomingConn *net.UDPConn
	outgoingAddr *net.UDPAddr
	ctx          context.Context
	cancel       context.CancelFunc
}

func (u *UDPBridge) Startup() error {
	u.ctx, u.cancel = context.WithCancel(u.Context)

	incomingAddr, err := net.ResolveUDPAddr("udp4", "localhost:4000")
	if err != nil {
		return fmt.Errorf("failed to resolve incoming UDP address: %w", err)
	}

	u.incomingConn, err = net.ListenUDP("udp4", incomingAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP address: %w", err)
	}

	u.outgoingAddr, err = net.ResolveUDPAddr("udp4", "localhost:4001")
	if err != nil {
		return fmt.Errorf("failed to resolve outgoing UDP address: %w", err)
	}

	go u.handleIncoming()

	if err = u.Nats.Subscribe("udp.outgoing", u.handleOutgoing); err != nil {
		return fmt.Errorf("failed to subscribe to udp.outgoing: %w", err)
	}

	return nil
}

func (u *UDPBridge) Shutdown() error {
	u.cancel()
	u.close(u.incomingConn)
	return nil
}

func (u *UDPBridge) handleIncoming() {
	incomingBuffer := make([]byte, 65535) // Max UDP packet size
	for {
		select {
		case <-u.ctx.Done():
			return
		default:
			n, _, err := u.incomingConn.ReadFromUDP(incomingBuffer)
			if err != nil {
				// Check if we're shutting down
				select {
				case <-u.ctx.Done():
					return
				default:
					u.Error("Error reading from UDP", "error", err)
					continue
				}
			}

			msg := nats.NewMsg("udp.incoming")
			msg.Data = make([]byte, n)
			copy(msg.Data, incomingBuffer[:n])
			u.Nats.PublishMsg(msg)
		}
	}
}

func (u *UDPBridge) close(c io.Closer) {
	if err := c.Close(); err != nil {
		u.Error("error closing connection", "error", err)
	}
}

func (u *UDPBridge) handleOutgoing(msg *nats.Msg) {
	if u.outgoingAddr == nil {
		u.Error("outgoing UDP address not configured")
		return
	}

	conn, err := net.DialUDP("udp", nil, u.outgoingAddr)
	if err != nil {
		u.Error("failed to create outgoing UDP connection", "error", err)
		return
	}
	defer u.close(conn)

	if _, err = conn.Write(msg.Data); err != nil {
		u.Error("failed to send UDP packet", "error", err)
	}
}
