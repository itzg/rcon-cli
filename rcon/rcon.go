// Package rcon implements a client for the Source RCON protocol, as spoken by
// Minecraft and other game servers.
//
// See https://minecraft.wiki/w/RCON and
// https://developer.valvesoftware.com/wiki/Source_RCON_Protocol
package rcon

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// Packet types. The protocol uses the same value for a command request and an
// authentication response.
const (
	typeResponse     int32 = 0
	typeCommand      int32 = 2
	typeAuthResponse int32 = 2
	typeAuth         int32 = 3
)

const (
	// headerSize is the size of the request ID and type fields.
	headerSize = 8
	// paddingSize is the size of the two null bytes that terminate a packet.
	paddingSize = 2
	// maxPacketSize bounds the size a server may declare for a single packet.
	// Servers are supposed to stay within 4096 bytes and fragment anything
	// longer, but there is no harm in being lenient.
	maxPacketSize = 64 * 1024

	// authFailedID is the request ID a server responds with when it rejects
	// the password.
	authFailedID int32 = -1

	// defaultTerminatorTimeout is how long to wait for more of a response
	// before concluding that the server does not answer the terminator packet.
	defaultTerminatorTimeout = time.Second
)

// ErrAuthFailed is returned by Dial when the server rejects the password.
var ErrAuthFailed = errors.New("rcon: authentication failed")

// Client is an authenticated connection to an RCON server. It is not safe for
// concurrent use.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	lastID int32

	terminatorTimeout time.Duration
	// noTerminator is set once the server has failed to answer a terminator
	// packet, see Execute.
	noTerminator bool
}

type packet struct {
	id   int32
	kind int32
	body string
}

// Dial connects to the RCON server at the given address and authenticates
// with the given password.
func Dial(address string, password string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	c := &Client{
		conn: conn,
		// Sized to hold a whole packet, since read relies on Peek
		reader:            bufio.NewReaderSize(conn, 4+maxPacketSize),
		terminatorTimeout: defaultTerminatorTimeout,
	}

	if err := c.authenticate(password); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return c, nil
}

// Close closes the connection to the server.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Execute sends the command to the server and returns its response.
//
// A response that does not fit in one packet gets split by the server over
// several packets, without any indication of which one is the last. To find
// the end, a second packet is sent after the command and the response is read
// until the reply to that one shows up, since servers process requests in
// order. An empty packet of the response type is used for that: Minecraft
// answers it with an "Unknown request" response and Source servers echo it
// back, without either of them executing anything.
//
// Not every server answers such a packet, so the wait for it is bounded. When
// it runs out, whatever was received up to then is returned, and the remaining
// commands on this connection settle for the first packet of the response
// rather than paying for that wait again. Likewise, losing the connection
// after the response has started, such as when stopping the server, is not
// treated as an error.
func (c *Client) Execute(command string) (string, error) {
	id := c.nextID()
	if err := c.write(id, typeCommand, command); err != nil {
		return "", err
	}

	var response strings.Builder
	for {
		// Packets with any other ID are leftovers from earlier requests
		p, err := c.read()
		if err != nil {
			return "", err
		}
		if p.id == id {
			response.WriteString(p.body)
			break
		}
	}

	if c.noTerminator {
		return response.String(), nil
	}

	// The terminator is deliberately sent only now that the server has started
	// responding, rather than along with the command. The Minecraft server
	// expects each read from the socket to give it exactly one packet and
	// closes the connection otherwise, which is what two packets written back
	// to back tend to cause.
	terminatorID := c.nextID()
	if err := c.write(terminatorID, typeResponse, ""); err != nil {
		return response.String(), nil
	}

	defer func() {
		_ = c.conn.SetReadDeadline(time.Time{})
	}()
	for {
		// The timeout starts over with each packet, so that it does not need
		// to cover the transfer of a long response.
		_ = c.conn.SetReadDeadline(time.Now().Add(c.terminatorTimeout))

		p, err := c.read()
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				c.noTerminator = true
			}
			return response.String(), nil
		}

		switch p.id {
		case terminatorID:
			return response.String(), nil
		case id:
			response.WriteString(p.body)
		}
	}
}

func (c *Client) authenticate(password string) error {
	id := c.nextID()
	if err := c.write(id, typeAuth, password); err != nil {
		return err
	}

	for {
		p, err := c.read()
		if err != nil {
			return err
		}
		// Source servers send an empty response packet ahead of the actual
		// authentication response
		if p.kind != typeAuthResponse {
			continue
		}
		if p.id == authFailedID {
			return ErrAuthFailed
		}
		if p.id == id {
			return nil
		}
	}
}

func (c *Client) nextID() int32 {
	c.lastID++
	return c.lastID
}

// write sends a packet using a single write to the connection, since the
// Minecraft server does not cope with a packet arriving in pieces.
func (c *Client) write(id int32, kind int32, body string) error {
	size := headerSize + len(body) + paddingSize

	buf := make([]byte, 0, 4+size)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(size))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(id))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(kind))
	buf = append(buf, body...)
	buf = append(buf, 0, 0)

	_, err := c.conn.Write(buf)
	return err
}

// read reads the next packet from the connection. Nothing is consumed until
// the whole packet has arrived, so reading can resume after a timeout.
func (c *Client) read() (packet, error) {
	buf, err := c.reader.Peek(4)
	if err != nil {
		return packet{}, err
	}

	size := int(int32(binary.LittleEndian.Uint32(buf)))
	if size < headerSize+paddingSize || size > maxPacketSize {
		return packet{}, fmt.Errorf("rcon: invalid packet size %d", size)
	}

	buf, err = c.reader.Peek(4 + size)
	if err != nil {
		return packet{}, err
	}

	p := packet{
		id:   int32(binary.LittleEndian.Uint32(buf[4:])),
		kind: int32(binary.LittleEndian.Uint32(buf[8:])),
		body: string(buf[4+headerSize : 4+size-paddingSize]),
	}

	_, err = c.reader.Discard(4 + size)
	return p, err
}
