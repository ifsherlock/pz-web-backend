package gamequery

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestParseInfoResponse(t *testing.T) {
	response := infoResponse(3, 8)
	count, err := parseInfoResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if count.Online != 3 || count.Max != 8 {
		t.Fatalf("count=%+v, want 3/8", count)
	}
}

func TestParseInfoResponseRejectsTruncatedPacket(t *testing.T) {
	if _, err := parseInfoResponse([]byte("\xff\xff\xff\xffI\x11server")); err == nil {
		t.Fatal("expected truncated response error")
	}
}

func TestQueryPlayerCountHandlesChallenge(t *testing.T) {
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	done := make(chan error, 1)
	go func() {
		buffer := make([]byte, 2048)
		address := net.Addr(nil)
		for step := range 2 {
			n, peer, readErr := server.ReadFrom(buffer)
			if readErr != nil {
				done <- readErr
				return
			}
			address = peer
			if step == 0 {
				_, readErr = server.WriteTo([]byte{'\xff', '\xff', '\xff', '\xff', 'A', 1, 2, 3, 4}, address)
			} else {
				if !bytes.HasSuffix(buffer[:n], []byte{1, 2, 3, 4}) {
					done <- context.Canceled
					return
				}
				_, readErr = server.WriteTo(infoResponse(2, 16), address)
			}
			if readErr != nil {
				done <- readErr
				return
			}
		}
		done <- nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	count, err := QueryPlayerCount(ctx, server.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	if count.Online != 2 || count.Max != 16 {
		t.Fatalf("count=%+v, want 2/16", count)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func infoResponse(online, max byte) []byte {
	response := []byte{'\xff', '\xff', '\xff', '\xff', 'I', 17}
	for _, value := range []string{"Test Server", "Muldraugh, KY", "zomboid", "Project Zomboid"} {
		response = append(response, value...)
		response = append(response, 0)
	}
	appID := make([]byte, 2)
	binary.LittleEndian.PutUint16(appID, 380870&0xffff)
	response = append(response, appID...)
	response = append(response, online, max, 0, 'd', 'l', 0, 1)
	return append(response, []byte("1.0.0.0\x00")...)
}
