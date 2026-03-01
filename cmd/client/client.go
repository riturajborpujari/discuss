package main

import (
	"fmt"
	"bufio"
	"net"
	"os"
)

const (
	Port = "6060"
)

func displayServerMessages(conn net.Conn) {
	buf := make([]byte, 512)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			return
		}
		fmt.Printf("\x1B[38;2;100;150;255m%s\x1B[0m\n", string(buf[0:n]))
	}
}

func main() {
	host := "localhost"
	if len(os.Args) > 1 {
		host = os.Args[1]
	}

	address := net.JoinHostPort(host, Port)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Could not connect %s: %v\n", address, err)
		os.Exit(1)
	}
	go displayServerMessages(conn)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		conn.Write(line)
	}
}
