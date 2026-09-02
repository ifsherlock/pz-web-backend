package gamequery

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

var infoRequest = []byte("\xff\xff\xff\xffTSource Engine Query\x00")

// PlayerCount is the live player count returned by the game's A2S_INFO endpoint.
type PlayerCount struct {
	Online int
	Max    int
}

// QueryPlayerCount reads the live player count from a Project Zomboid UDP query port.
func QueryPlayerCount(ctx context.Context, address string) (PlayerCount, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, "udp", address)
	if err != nil {
		return PlayerCount{}, err
	}
	defer conn.Close()

	deadline := time.Now().Add(time.Second)
	if value, ok := ctx.Deadline(); ok && value.Before(deadline) {
		deadline = value
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return PlayerCount{}, err
	}

	response, err := exchange(conn, infoRequest)
	if err != nil {
		return PlayerCount{}, err
	}
	if len(response) >= 9 && bytes.Equal(response[:5], []byte{'\xff', '\xff', '\xff', '\xff', 'A'}) {
		request := append(append([]byte{}, infoRequest...), response[5:9]...)
		response, err = exchange(conn, request)
		if err != nil {
			return PlayerCount{}, err
		}
	}

	return parseInfoResponse(response)
}

func exchange(conn net.Conn, request []byte) ([]byte, error) {
	if _, err := conn.Write(request); err != nil {
		return nil, err
	}
	buffer := make([]byte, 64*1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

func parseInfoResponse(data []byte) (PlayerCount, error) {
	if len(data) < 6 || !bytes.Equal(data[:5], []byte{'\xff', '\xff', '\xff', '\xff', 'I'}) {
		return PlayerCount{}, fmt.Errorf("unexpected A2S_INFO response")
	}

	position := 6 // header, response type and protocol byte
	for range 4 { // server name, map, folder and game name
		end := bytes.IndexByte(data[position:], 0)
		if end < 0 {
			return PlayerCount{}, fmt.Errorf("truncated A2S_INFO response")
		}
		position += end + 1
	}
	if len(data) < position+5 {
		return PlayerCount{}, fmt.Errorf("truncated A2S_INFO player fields")
	}

	_ = binary.LittleEndian.Uint16(data[position : position+2]) // app ID
	return PlayerCount{Online: int(data[position+2]), Max: int(data[position+3])}, nil
}
