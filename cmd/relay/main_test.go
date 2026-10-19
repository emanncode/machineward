package main

import (
	"bytes"
	"net"
	"strings"
	"testing"
)

func TestListenDoesNotReportSuccessWhenAddressIsOccupied(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()

	var output bytes.Buffer
	listener, err := listen(occupied.Addr().String(), &output)
	if err == nil {
		if listener != nil {
			_ = listener.Close()
		}
		t.Fatal("listen succeeded while the address was occupied")
	}
	if listener != nil {
		t.Fatal("listen returned a listener after the bind failed")
	}
	if output.Len() != 0 {
		t.Fatalf("startup output = %q, want no listening message", output.String())
	}
}

func TestListenReportsSuccessAfterBinding(t *testing.T) {
	var output bytes.Buffer
	listener, err := listen("127.0.0.1:0", &output)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	if !strings.Contains(output.String(), "Machineward local relay listening at ws://127.0.0.1:0/ws") {
		t.Fatalf("startup output = %q, want listening message", output.String())
	}
}
