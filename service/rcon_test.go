package service

import (
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/mlctrez/rcon"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRconServer is a minimal Factorio-like RCON server. It accepts any
// password and answers each command with respond(command).
type fakeRconServer struct {
	listener net.Listener
	respond  func(command string) string

	mu       sync.Mutex
	received []string
}

func newFakeRconServer(t *testing.T, respond func(string) string) *fakeRconServer {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &fakeRconServer{listener: l, respond: respond}
	go s.serve()
	t.Cleanup(func() { _ = l.Close() })
	return s
}

func (s *fakeRconServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeRconServer) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	auth := &rcon.Packet{}
	if _, err := auth.ReadFrom(conn); err != nil {
		return
	}
	if _, err := rcon.NewPacket(rcon.SERVERDATA_AUTH_RESPONSE, auth.ID, "").WriteTo(conn); err != nil {
		return
	}
	for {
		req := &rcon.Packet{}
		if _, err := req.ReadFrom(conn); err != nil {
			return
		}
		s.mu.Lock()
		s.received = append(s.received, req.Body())
		s.mu.Unlock()
		if _, err := rcon.NewPacket(rcon.SERVERDATA_RESPONSE_VALUE, req.ID, s.respond(req.Body())).WriteTo(conn); err != nil {
			return
		}
	}
}

func (s *fakeRconServer) commands() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.received...)
}

// newTestRcon starts an in-process NATS server on a random port and an Rcon
// component connected to a fake RCON server, with its subscriptions active.
// It returns a client NATS connection for sending requests.
func newTestRcon(t *testing.T, fake *fakeRconServer) *nats.Conn {
	t.Helper()
	logger := *slog.New(slog.NewTextHandler(io.Discard, nil))

	ns, err := server.NewServer(&server.Options{Host: "127.0.0.1", Port: -1, NoSigs: true, NoLog: true})
	require.NoError(t, err)
	go ns.Start()
	require.True(t, ns.ReadyForConnections(5*time.Second))
	t.Cleanup(ns.Shutdown)

	serviceConn, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	t.Cleanup(serviceConn.Close)

	r := &Rcon{Logger: logger, Nats: &Nats{Logger: logger, conn: serviceConn}}
	require.NoError(t, r.Startup())
	if fake != nil {
		r.conn, err = rcon.Dial(fake.listener.Addr().String(), "pw", rcon.SetExeCommandId(10))
		require.NoError(t, err)
		t.Cleanup(func() { _ = r.Shutdown() })
	}

	client, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client
}

const playersOutput = "Players (2):\n  alice (online)\n2026-10-23 12:00:00 [JOIN] bob joined the game\n  bob (online)\n"

const whitelistOutput = "Whitelisted players (2):\n  alice\n2026-10-23 12:00:00 [JOIN] bob joined the game\n  bob\n"

func fakeFactorioResponse(command string) string {
	switch command {
	case "/players":
		return playersOutput
	case "/whitelist get":
		return whitelistOutput
	case "/bans":
		return "There are no banned players."
	case "/sc rcon.print(game.tick)":
		return "12345\n"
	default:
		return "unknown command"
	}
}

func request(t *testing.T, nc *nats.Conn, subject, data string) (string, string) {
	t.Helper()
	resp, err := nc.Request(subject, []byte(data), 5*time.Second)
	require.NoError(t, err)
	return string(resp.Data), resp.Header.Get("error")
}

func TestRconConsoleForwardsVerbatimAndReturnsAllLines(t *testing.T) {
	fake := newFakeRconServer(t, fakeFactorioResponse)
	nc := newTestRcon(t, fake)

	data, errHdr := request(t, nc, ConsoleSubject, "/whitelist get\n")
	assert.Empty(t, errHdr)
	assert.Equal(t, "Whitelisted players (2):\n  alice\n  bob", data, "all lines except console messages")

	data, errHdr = request(t, nc, ConsoleSubject, "  /bans  ")
	assert.Empty(t, errHdr)
	assert.Equal(t, "There are no banned players.", data)

	assert.Equal(t, []string{"/whitelist get", "/bans"}, fake.commands(), "sent verbatim with no Lua wrapping")
}

func TestRconConsoleRejectsInvalidCommands(t *testing.T) {
	fake := newFakeRconServer(t, fakeFactorioResponse)
	nc := newTestRcon(t, fake)

	for _, bad := range []string{"", "   ", "hello everyone", "/bans\n/mutes", "/players", "/server-save", "/kick bob", "/sc rcon.print(1)", "/c game.print(1)"} {
		_, errHdr := request(t, nc, ConsoleSubject, bad)
		assert.NotEmpty(t, errHdr, "input %q should be rejected", bad)
	}
	assert.Empty(t, fake.commands(), "rejected commands must not reach RCON")
}

func TestRconConsoleNotConnected(t *testing.T) {
	nc := newTestRcon(t, nil)
	_, errHdr := request(t, nc, ConsoleSubject, "/bans")
	assert.Equal(t, "rcon not connected", errHdr)
}

func TestRconHandleUnchanged(t *testing.T) {
	fake := newFakeRconServer(t, fakeFactorioResponse)
	nc := newTestRcon(t, fake)

	data, errHdr := request(t, nc, "factop.rcon", "/sc rcon.print(game.tick)")
	assert.Empty(t, errHdr)
	assert.Equal(t, "12345", data)

	// factop.rcon still returns only the last non-console line.
	data, errHdr = request(t, nc, "factop.rcon", "/players")
	assert.Empty(t, errHdr)
	assert.Equal(t, "  bob (online)", data)

	assert.Equal(t, []string{"/sc rcon.print(game.tick)", "/players"}, fake.commands())
}

func TestConsoleCommand(t *testing.T) {
	// Full validation cases live in the console package; this checks the
	// service uses it.
	got, err := ConsoleCommand(" /whitelist get \n")
	require.NoError(t, err)
	assert.Equal(t, "/whitelist get", got)

	_, err = ConsoleCommand("/players")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestRconFilterOutput(t *testing.T) {
	r := &Rcon{Logger: *slog.New(slog.NewTextHandler(io.Discard, nil))}
	assert.Equal(t, []string{"Players (2):", "  alice (online)", "  bob (online)"}, r.filterOutput(playersOutput))
	assert.Empty(t, r.filterOutput(""))
	assert.Empty(t, r.filterOutput("2026-10-23 12:00:00 [CHAT] alice: hi\n"))
}
