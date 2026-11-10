package rcon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

const testPassword = "secret"

// fakeServer is an RCON server that behaves like the one in Minecraft: a
// response gets split into packets of up to 4096 bytes, a packet of an unknown
// type gets an "Unknown request" response, and each read from the socket needs
// to yield exactly one packet.
type fakeServer struct {
	// handler provides the response to a command
	handler func(command string) string
	// ignoreUnknown leaves packets of an unknown type unanswered
	ignoreUnknown bool
	// closeAfterResponse closes the connection once a command got responded to
	closeAfterResponse bool
	// readDelay is waited before each read, giving packets that were wrongly
	// sent ahead of time the chance to pile up
	readDelay time.Duration

	listener net.Listener
	wg       sync.WaitGroup

	mu         sync.Mutex
	conns      []net.Conn
	commands   []string
	unknown    int
	violations int
}

func startFakeServer(t *testing.T, server *fakeServer) *fakeServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server.listener = listener

	server.wg.Add(1)
	go server.accept()

	t.Cleanup(func() {
		_ = listener.Close()
		server.mu.Lock()
		for _, conn := range server.conns {
			_ = conn.Close()
		}
		server.mu.Unlock()
		server.wg.Wait()

		if server.violations > 0 {
			t.Errorf("server got %d reads that were not exactly one packet", server.violations)
		}
	})

	return server
}

func (s *fakeServer) dial(t *testing.T) *Client {
	t.Helper()

	client, err := Dial(s.listener.Addr().String(), testPassword)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	return client
}

func (s *fakeServer) accept() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}

		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()

		s.wg.Add(1)
		go s.serve(conn)
	}
}

func (s *fakeServer) serve(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	authenticated := false
	buf := make([]byte, 1460)
	for {
		time.Sleep(s.readDelay)

		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		if n < 14 || int(binary.LittleEndian.Uint32(buf)) != n-4 {
			s.mu.Lock()
			s.violations++
			s.mu.Unlock()
			return
		}

		id := int32(binary.LittleEndian.Uint32(buf[4:]))
		kind := int32(binary.LittleEndian.Uint32(buf[8:]))
		body := string(buf[12 : n-2])

		switch {
		case kind == typeAuth:
			authenticated = body == testPassword
			if authenticated {
				writePacket(conn, id, typeAuthResponse, "")
			} else {
				writePacket(conn, authFailedID, typeAuthResponse, "")
			}

		case !authenticated:
			writePacket(conn, authFailedID, typeAuthResponse, "")

		case kind == typeCommand:
			s.mu.Lock()
			s.commands = append(s.commands, body)
			s.mu.Unlock()

			response := s.handler(body)
			for {
				chunk := response[:min(len(response), 4096)]
				writePacket(conn, id, typeResponse, chunk)

				response = response[len(chunk):]
				if response == "" {
					break
				}
			}
			if s.closeAfterResponse {
				return
			}

		default:
			s.mu.Lock()
			s.unknown++
			s.mu.Unlock()

			if !s.ignoreUnknown {
				writePacket(conn, id, typeResponse, fmt.Sprintf("Unknown request %x", kind))
			}
		}
	}
}

func (s *fakeServer) unknownCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.unknown
}

func writePacket(conn net.Conn, id int32, kind int32, body string) {
	buf := make([]byte, 0, 14+len(body))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(10+len(body)))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(id))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(kind))
	buf = append(buf, body...)
	buf = append(buf, 0, 0)
	_, _ = conn.Write(buf)
}

// longResponse builds a response of numbered lines that takes several packets
func longResponse(label string) string {
	var sb strings.Builder
	for i := 0; sb.Len() < 3*4096; i++ {
		_, _ = fmt.Fprintf(&sb, "%s line %d\n", label, i)
	}
	return sb.String()
}

func respond(command string) string {
	if strings.HasPrefix(command, "long") {
		return longResponse(command)
	}
	return "ran " + command
}

func assertExecute(t *testing.T, client *Client, command string, expected string) {
	t.Helper()

	response, err := client.Execute(command)
	if err != nil {
		t.Fatalf("executing %q: %v", command, err)
	}
	if response != expected {
		t.Fatalf("executing %q: got %d bytes %.40q..., expected %d bytes %.40q...",
			command, len(response), response, len(expected), expected)
	}
}

func TestDial_wrongPassword(t *testing.T) {
	server := startFakeServer(t, &fakeServer{handler: respond})

	_, err := Dial(server.listener.Addr().String(), "wrong")
	if !errors.Is(err, ErrAuthFailed) {
		t.Fatalf("got %v, expected %v", err, ErrAuthFailed)
	}
}

func TestExecute_singlePacket(t *testing.T) {
	server := startFakeServer(t, &fakeServer{handler: respond})
	client := server.dial(t)

	assertExecute(t, client, "list", "ran list")
}

func TestExecute_emptyResponse(t *testing.T) {
	server := startFakeServer(t, &fakeServer{handler: func(string) string { return "" }})
	client := server.dial(t)

	assertExecute(t, client, "save-on", "")
}

func TestExecute_multiplePackets(t *testing.T) {
	server := startFakeServer(t, &fakeServer{handler: respond})
	client := server.dial(t)

	assertExecute(t, client, "long", longResponse("long"))
}

// The response to a command needs to be consumed entirely for the next command
// to not end up with what was left of it.
func TestExecute_consecutiveCommands(t *testing.T) {
	server := startFakeServer(t, &fakeServer{handler: respond})
	client := server.dial(t)

	assertExecute(t, client, "long 1", longResponse("long 1"))
	assertExecute(t, client, "list", "ran list")
	assertExecute(t, client, "long 2", longResponse("long 2"))
	assertExecute(t, client, "seed", "ran seed")
}

// With the server being slow to read, a terminator that got sent along with
// the command would reach it in the same read and be counted as a violation.
func TestExecute_terminatorNotSentAlongWithCommand(t *testing.T) {
	server := startFakeServer(t, &fakeServer{handler: respond, readDelay: 20 * time.Millisecond})
	client := server.dial(t)

	assertExecute(t, client, "list", "ran list")
	assertExecute(t, client, "long", longResponse("long"))

	if count := server.unknownCount(); count != 2 {
		t.Fatalf("server got %d terminators, expected 2", count)
	}
}

func TestExecute_terminatorNotAnswered(t *testing.T) {
	server := startFakeServer(t, &fakeServer{handler: respond, ignoreUnknown: true})
	client := server.dial(t)
	client.terminatorTimeout = 100 * time.Millisecond

	// Everything that showed up while waiting for the terminator is used
	start := time.Now()
	assertExecute(t, client, "long 1", longResponse("long 1"))
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("took %s to give up on the terminator", elapsed)
	}

	// From then on only the first packet of a response is used, with the rest
	// of it being skipped rather than taken for the next response
	assertExecute(t, client, "list", "ran list")
	assertExecute(t, client, "long 2", longResponse("long 2")[:4096])
	assertExecute(t, client, "seed", "ran seed")

	if count := server.unknownCount(); count != 1 {
		t.Fatalf("server got %d terminators, expected 1", count)
	}
}

// A server that stops will respond to the command, but might be gone before
// answering the terminator.
func TestExecute_connectionClosedAfterResponse(t *testing.T) {
	server := startFakeServer(t, &fakeServer{handler: respond, closeAfterResponse: true})
	client := server.dial(t)

	assertExecute(t, client, "stop", "ran stop")

	if _, err := client.Execute("list"); err == nil {
		t.Fatal("expected an error with the connection closed")
	}
}
