// Derived from github.com/gorcon/rcon (via github.com/mlctrez/rcon),
// Copyright (c) 2020 Pavel Korotkiy (outdead), MIT License. See LICENSE.

package rcon_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mlctrez/factop/internal/rcon"
	"github.com/mlctrez/factop/internal/rcon/rcontest"
)

func authHandler(c *rcontest.Context) {
	switch c.Request().Body() {
	case "invalid packet type":
		rcon.NewPacket(42, c.Request().ID, "").WriteTo(c.Conn())
	case "another":
		rcon.NewPacket(rcon.SERVERDATA_AUTH_RESPONSE, 42, "").WriteTo(c.Conn())
	case "makeslice":
		size := int32(len([]byte("")))

		buffer := bytes.NewBuffer(make([]byte, 0, size+4))

		_ = binary.Write(buffer, binary.LittleEndian, size)
		_ = binary.Write(buffer, binary.LittleEndian, c.Request().ID)
		_ = binary.Write(buffer, binary.LittleEndian, rcon.SERVERDATA_RESPONSE_VALUE)

		buffer.WriteTo(c.Conn())
	case c.Server().Settings.Password:
		rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, c.Request().ID, "").WriteTo(c.Conn())
		rcon.NewPacket(rcon.SERVERDATA_AUTH_RESPONSE, c.Request().ID, "").WriteTo(c.Conn())
	default:
		rcon.NewPacket(rcon.SERVERDATA_AUTH_RESPONSE, -1, string([]byte{0x00})).WriteTo(c.Conn())
	}
}

func commandHandler(c *rcontest.Context) {
	writeWithInvalidPadding := func(conn io.Writer, packet *rcon.Packet) {
		buffer := bytes.NewBuffer(make([]byte, 0, packet.Size+4))

		binary.Write(buffer, binary.LittleEndian, packet.Size)
		binary.Write(buffer, binary.LittleEndian, packet.ID)
		binary.Write(buffer, binary.LittleEndian, packet.Type)

		// Write command body, null terminated ASCII string and an empty ASCIIZ string.
		// Second padding byte is incorrect.
		buffer.Write(append([]byte(packet.Body()), 0x00, 0x01))

		buffer.WriteTo(conn)
	}

	switch c.Request().Body() {
	case "help":
		responseBody := "lorem ipsum dolor sit amet"
		rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, c.Request().ID, responseBody).WriteTo(c.Conn())
	case "rust":
		// Write specific Rust package.
		rcon.NewPacket(4, c.Request().ID, "").WriteTo(c.Conn())

		rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, -1, c.Request().Body()).WriteTo(c.Conn())
	case "padding":
		writeWithInvalidPadding(c.Conn(), rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, c.Request().ID, ""))
	case "another":
		rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, 42, "").WriteTo(c.Conn())
	default:
		rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, c.Request().ID, "unknown command").WriteTo(c.Conn())
	}
}

func TestDial(t *testing.T) {
	server := rcontest.NewServer(rcontest.SetSettings(rcontest.Settings{Password: "password"}))
	defer server.Close()

	t.Run("connection refused", func(t *testing.T) {
		wantErrContains := "connect: connection refused"

		_, err := rcon.Dial("127.0.0.2:12345", "password")
		if err == nil || !strings.Contains(err.Error(), wantErrContains) {
			t.Errorf("got err %q, want to contain %q", err, wantErrContains)
		}
	})

	t.Run("connection timeout", func(t *testing.T) {
		server := rcontest.NewServer(rcontest.SetSettings(rcontest.Settings{Password: "password", AuthResponseDelay: 6 * time.Second}))
		defer server.Close()

		wantErrContains := "i/o timeout"

		_, err := rcon.Dial(server.Addr(), "", rcon.SetDialTimeout(5*time.Second))
		if err == nil || !strings.Contains(err.Error(), wantErrContains) {
			t.Errorf("got err %q, want to contain %q", err, wantErrContains)
		}
	})

	t.Run("authentication failed", func(t *testing.T) {
		_, err := rcon.Dial(server.Addr(), "wrong")
		if !errors.Is(err, rcon.ErrAuthFailed) {
			t.Errorf("got err %q, want %q", err, rcon.ErrAuthFailed)
		}
	})

	t.Run("invalid packet type", func(t *testing.T) {
		server := rcontest.NewServer(
			rcontest.SetSettings(rcontest.Settings{Password: "password"}),
			rcontest.SetAuthHandler(authHandler),
		)
		defer server.Close()

		_, err := rcon.Dial(server.Addr(), "invalid packet type")
		if !errors.Is(err, rcon.ErrInvalidAuthResponse) {
			t.Errorf("got err %q, want %q", err, rcon.ErrInvalidAuthResponse)
		}
	})

	t.Run("invalid response id", func(t *testing.T) {
		server := rcontest.NewServer(
			rcontest.SetSettings(rcontest.Settings{Password: "password"}),
			rcontest.SetAuthHandler(authHandler),
		)
		defer server.Close()

		_, err := rcon.Dial(server.Addr(), "another")
		if !errors.Is(err, rcon.ErrInvalidPacketID) {
			t.Errorf("got err %q, want %q", err, rcon.ErrInvalidPacketID)
		}
	})

	t.Run("makeslice", func(t *testing.T) {
		server := rcontest.NewServer(
			rcontest.SetSettings(rcontest.Settings{Password: "makeslice"}),
			rcontest.SetAuthHandler(authHandler),
		)
		defer server.Close()

		_, err := rcon.Dial(server.Addr(), "makeslice")
		if !errors.Is(err, rcon.ErrAuthNotRCON) {
			t.Errorf("got err %q, want %q", err, rcon.ErrAuthNotRCON)
		}
	})

	t.Run("auth success", func(t *testing.T) {
		conn, err := rcon.Dial(server.Addr(), "password")
		if err != nil {
			t.Errorf("got err %q, want %v", err, nil)
			return
		}

		conn.Close()
	})
}

func TestConn_Execute(t *testing.T) {
	server := rcontest.NewUnstartedServer()
	server.Settings.Password = "password"
	server.SetCommandHandler(commandHandler)
	server.Start()
	defer server.Close()

	t.Run("incorrect command", func(t *testing.T) {
		conn, err := rcon.Dial(server.Addr(), "password")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		defer conn.Close()

		result, err := conn.Execute("")
		if !errors.Is(err, rcon.ErrCommandEmpty) {
			t.Errorf("got err %q, want %q", err, rcon.ErrCommandEmpty)
		}

		if len(result) != 0 {
			t.Fatalf("got result len %d, want %d", len(result), 0)
		}

		result, err = conn.Execute(string(make([]byte, 1001)))
		if !errors.Is(err, rcon.ErrCommandTooLong) {
			t.Errorf("got err %q, want %q", err, rcon.ErrCommandTooLong)
		}

		if len(result) != 0 {
			t.Fatalf("got result len %d, want %d", len(result), 0)
		}
	})

	t.Run("closed network connection 1", func(t *testing.T) {
		conn, err := rcon.Dial(server.Addr(), "password", rcon.SetDeadline(0))
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		conn.Close()

		result, err := conn.Execute("help")
		wantErrMsg := fmt.Sprintf("write tcp %s->%s: use of closed network connection", conn.LocalAddr(), conn.RemoteAddr())
		if err == nil || err.Error() != wantErrMsg {
			t.Errorf("got err %q, want to contain %q", err, wantErrMsg)
		}

		if len(result) != 0 {
			t.Fatalf("got result len %d, want %d", len(result), 0)
		}
	})

	t.Run("closed network connection 2", func(t *testing.T) {
		conn, err := rcon.Dial(server.Addr(), "password")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		conn.Close()

		result, err := conn.Execute("help")
		wantErrMsg := fmt.Sprintf("rcon: set tcp %s: use of closed network connection", conn.LocalAddr())
		if err == nil || err.Error() != wantErrMsg {
			t.Errorf("got err %q, want to contain %q", err, wantErrMsg)
		}

		if len(result) != 0 {
			t.Fatalf("got result len %d, want %d", len(result), 0)
		}
	})

	t.Run("read deadline", func(t *testing.T) {
		server := rcontest.NewServer(rcontest.SetSettings(rcontest.Settings{Password: "password", CommandResponseDelay: 2 * time.Second}))
		defer server.Close()

		conn, err := rcon.Dial(server.Addr(), "password", rcon.SetDeadline(1*time.Second))
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		defer conn.Close()

		result, err := conn.Execute("deadline")
		wantErrMsg := fmt.Sprintf("rcon: read packet size: read tcp %s->%s: i/o timeout", conn.LocalAddr(), conn.RemoteAddr())
		if err == nil || err.Error() != wantErrMsg {
			t.Errorf("got err %q, want to contain %q", err, wantErrMsg)
		}

		if len(result) != 0 {
			t.Fatalf("got result len %d, want %d", len(result), 0)
		}
	})

	t.Run("invalid padding", func(t *testing.T) {
		conn, err := rcon.Dial(server.Addr(), "password")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		defer conn.Close()

		result, err := conn.Execute("padding")
		if !errors.Is(err, rcon.ErrInvalidPacketPadding) {
			t.Errorf("got err %q, want %q", err, rcon.ErrInvalidPacketPadding)
		}

		if len(result) != 2 {
			t.Fatalf("got result len %d, want %d", len(result), 2)
		}
	})

	t.Run("invalid response id", func(t *testing.T) {
		conn, err := rcon.Dial(server.Addr(), "password")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		defer conn.Close()

		result, err := conn.Execute("another")
		if !errors.Is(err, rcon.ErrInvalidPacketID) {
			t.Errorf("got err %q, want %q", err, rcon.ErrInvalidPacketID)
		}

		if len(result) != 0 {
			t.Fatalf("got result len %d, want %d", len(result), 0)
		}
	})

	t.Run("success help command", func(t *testing.T) {
		conn, err := rcon.Dial(server.Addr(), "password")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		defer conn.Close()

		result, err := conn.Execute("help")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}

		resultWant := "lorem ipsum dolor sit amet"
		if result != resultWant {
			t.Fatalf("got result %q, want %q", result, resultWant)
		}
	})

	t.Run("rust workaround", func(t *testing.T) {
		conn, err := rcon.Dial(server.Addr(), "password", rcon.SetDeadline(1*time.Second))
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}
		defer conn.Close()

		result, err := conn.Execute("rust")
		if err != nil {
			t.Fatalf("got err %q, want %v", err, nil)
		}

		resultWant := "rust"
		if result != resultWant {
			t.Fatalf("got result %q, want %q", result, resultWant)
		}
	})
}

func TestConn_ExecuteCommandID(t *testing.T) {
	var (
		mu     sync.Mutex
		gotIDs []int32
	)

	server := rcontest.NewServer(
		rcontest.SetSettings(rcontest.Settings{Password: "password"}),
		rcontest.SetCommandHandler(func(c *rcontest.Context) {
			mu.Lock()
			gotIDs = append(gotIDs, c.Request().ID)
			mu.Unlock()

			switch c.Request().Body() {
			case "rust":
				// Type 4 packet followed by a response with ID -1: the client
				// must map -1 to the configured exec command id.
				rcon.NewPacket(4, c.Request().ID, "").WriteTo(c.Conn())
				rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, -1, "rust").WriteTo(c.Conn())
			case "zero":
				// Response with the default id 0 is not ours when id is 10.
				rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, 0, "console").WriteTo(c.Conn())
			default:
				rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, c.Request().ID, "ok").WriteTo(c.Conn())
			}
		}),
	)
	defer server.Close()

	conn, err := rcon.Dial(server.Addr(), "password", rcon.SetExeCommandId(10))
	if err != nil {
		t.Fatalf("got err %q, want %v", err, nil)
	}
	defer conn.Close()

	result, err := conn.Execute("help")
	if err != nil || result != "ok" {
		t.Fatalf("got %q, %v, want %q, nil", result, err, "ok")
	}

	result, err = conn.Execute("rust")
	if err != nil || result != "rust" {
		t.Fatalf("got %q, %v, want %q, nil", result, err, "rust")
	}

	result, err = conn.Execute("zero")
	if !errors.Is(err, rcon.ErrInvalidPacketID) {
		t.Fatalf("got err %v, want %v", err, rcon.ErrInvalidPacketID)
	}
	if result != "console" {
		t.Fatalf("got result %q, want %q", result, "console")
	}

	mu.Lock()
	defer mu.Unlock()

	for i, id := range gotIDs {
		if id != 10 {
			t.Errorf("request %d sent with id %d, want 10", i, id)
		}
	}
	if len(gotIDs) != 3 {
		t.Errorf("got %d requests, want 3", len(gotIDs))
	}
}
