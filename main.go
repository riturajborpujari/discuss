package main

import (
	"fmt"
	"net"
	"os"
	"regexp"
)

type Client struct {
	Username string
	conn     net.Conn
}

type Message struct {
	Author string
	Text   []byte
}

const (
	Port              = "6060"
	BufferSize        = 512
	MessageBufferSize = 10
)

var (
	clients      map[string]Client = map[string]Client{}
	invalidChars *regexp.Regexp    = regexp.MustCompile("[^A-Za-z0-9_]")
	crlf         *regexp.Regexp    = regexp.MustCompile("[\r\n]+$")
)

func identifyClient(conn net.Conn) (Client, error) {
	buf := make([]byte, 32)
	chosenUsername := ""
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return Client{}, err
		}

		trimmedBuf := crlf.ReplaceAll(buf[0:n], []byte(""))
		if len(trimmedBuf) == 0 {
			conn.Write([]byte("Username can't be empty. Try again"))
			continue
		}
		if invalidChars.Match(trimmedBuf) == true {
			conn.Write([]byte("Username can only contain\n" +
				" - English alphabets\n" +
				" - Numbers\n" +
				" - Underscore(_)\n" +
				"Please try again"))
			continue
		}

		chosenUsername = string(trimmedBuf)
		_, exists := clients[chosenUsername]
		if exists == true {
			conn.Write(
				fmt.Appendf([]byte(""),
					"Username '%s' has been taken. Try another",
					chosenUsername))
			continue
		}
		clients[chosenUsername] = Client{
			chosenUsername,
			conn,
		}
		return clients[chosenUsername], nil
	}
}

func handleClientDisconnect(client Client) {
	delete(clients, client.Username)
	client.conn.Close()
}

func handleConn(conn net.Conn, messages chan<- Message) {
	conn.Write([]byte("Welcome to Chat! Please pick your username"))

	buf := make([]byte, BufferSize)
	client, err := identifyClient(conn)
	if err != nil {
		conn.Close()
		return
	}

	fmt.Printf("INFO: %v: Client connected\n", client.Username)
	client.conn.Write(
		fmt.Appendf([]byte(""),
			"Hello, %s. You can chat now",
			client.Username))
	messages <- Message{client.Username, []byte("has connected")}
	for {
		n, err := conn.Read(buf)
		if err != nil {
			messages <- Message{
				client.Username,
				[]byte("has disconnected"),
			}
			handleClientDisconnect(client)
			fmt.Printf("INFO: %v: Client Disconnected\n", client.Username)
			return
		}

		msgBytes := crlf.ReplaceAll(buf[0:n], []byte(""))
		if len(msgBytes) > 0 {
			messages <- Message{client.Username, msgBytes}
		}
	}
}

func handleMessages(messages <-chan Message) {
	for message := range messages {
		messageBytes := fmt.Appendf([]byte(""), "%s: %s", message.Author, message.Text)
		for _, client := range clients {
			if client.Username != message.Author {
				client.conn.Write(messageBytes)
			}
		}
	}
}

func main() {
	address := ":" + Port
	lis, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not listen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("INFO: Chat server started on %s\n", address)

	messages := make(chan Message, MessageBufferSize)
	go handleMessages(messages)
	for {
		conn, err := lis.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: could not accept: %v\n", err)
			continue
		}
		go handleConn(conn, messages)
	}
}
