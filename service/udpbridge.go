package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"

	"github.com/mlctrez/bind"
	"github.com/nats-io/nats.go"
)

var _ bind.Startup = (*UDPBridge)(nil)
var _ bind.Shutdown = (*UDPBridge)(nil)

// UDPBridge bridges factorio udp_send and udp_rec traffic to and from NATS
type UDPBridge struct {
	slog.Logger
	Context      context.Context
	Nats         *Nats
	Settings     *Settings
	incomingConn *net.UDPConn
	outgoingAddr *net.UDPAddr
	outgoingConn *net.UDPConn
}

func (u *UDPBridge) Startup() error {
	incomingAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("localhost:%d", u.Settings.Data.UDPIncoming))
	if err != nil {
		return fmt.Errorf("failed to resolve incoming UDP address: %w", err)
	}

	u.incomingConn, err = net.ListenUDP("udp4", incomingAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP address: %w", err)
	}

	u.outgoingAddr, err = net.ResolveUDPAddr("udp4", fmt.Sprintf("localhost:%d", u.Settings.Data.UDPOutgoing))
	if err != nil {
		return fmt.Errorf("failed to resolve outgoing UDP address: %w", err)
	}

	u.outgoingConn, err = net.DialUDP("udp4", nil, u.outgoingAddr)
	if err != nil {
		return fmt.Errorf("outgoing UDP connection: %w", err)
	}

	go u.handleIncoming()

	if err = u.Nats.Subscribe("udp.outgoing.>", u.handleOutgoing); err != nil {
		return fmt.Errorf("failed to subscribe to udp.outgoing: %w", err)
	}

	return nil
}

func (u *UDPBridge) Shutdown() error {
	u.close(u.incomingConn)
	u.close(u.outgoingConn)
	return nil
}

func (u *UDPBridge) handleIncoming() {
	buf := make([]byte, 65535)
	for {
		select {
		case <-u.Context.Done():
			return
		default:
		}

		n, _, err := u.incomingConn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-u.Context.Done():
				return
			default:
				u.Error("error reading UDP", "error", err)
				continue
			}
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		tag := extractTag(data)
		subject := "udp.incoming." + tag

		msg := nats.NewMsg(subject)
		msg.Data = data
		u.Nats.PublishMsg(msg)
	}
}

func (u *UDPBridge) close(c io.Closer) {
	if err := c.Close(); err != nil {
		u.Error("error closing connection", "error", err)
	}
}

func (u *UDPBridge) handleOutgoing(msg *nats.Msg) {
	if _, err := u.outgoingConn.Write(msg.Data); err != nil {
		u.Error("failed to send UDP", "error", err)
	}
}

// extractTag reads the content between the first [ and ] in the message.
// Returns "unknown" if no valid tag is found.
func extractTag(data []byte) string {
	if len(data) < 3 || data[0] != '[' {
		return "unknown"
	}
	end := bytes.IndexByte(data, ']')
	if end < 2 {
		return "unknown"
	}
	tag := string(data[1:end])
	return strings.ToLower(tag)
}
