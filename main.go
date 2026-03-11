package main

import (
	"fmt"
	"net"
	"os"
	"regexp"
)

type Client struct {
	Username  string
	SendQueue chan []byte
	Conn      net.Conn
}

type Message struct {
	Author string
	Text   []byte
}

const (
	Port                  = "6060"
	BufferSize            = 512
	MessageBufferSize     = 1024
	MsgQueueSizePerClient = 5
)

var (
	clients      map[string]Client = map[string]Client{}
	messages     chan Message      = make(chan Message, MessageBufferSize)
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
		client := Client{
			chosenUsername,
			make(chan []byte, MsgQueueSizePerClient),
			conn,
		}
		clients[chosenUsername] = client
		return client, nil
	}
}

func handleConn(conn net.Conn, messages chan<- Message) {
	conn.Write([]byte("Welcome to Chat! Please pick your username"))

	buf := make([]byte, BufferSize)
	client, err := identifyClient(conn)
	if err != nil {
		conn.Close()
		return
	}
	go clientWriter(client)

	fmt.Printf("INFO: %v: Client connected\n", client.Username)
	client.Conn.Write(
		fmt.Appendf([]byte(""),
			"Hello, %s. You can chat now",
			client.Username))
	messages <- Message{client.Username, []byte("has connected")}
	for {
		n, err := conn.Read(buf)
		if err != nil {
			handleClientDisconnect(client)
			return
		}

		msgBytes := crlf.ReplaceAll(buf[0:n], []byte(""))
		if len(msgBytes) > 0 {
			messages <- Message{client.Username, msgBytes}
		}
	}
}

func clientWriter(client Client) {
	for msg := range client.SendQueue {
		client.Conn.Write(msg)
	}
}

func handleClientDisconnect(client Client) {
	messages <- Message{
		client.Username,
		[]byte("has disconnected"),
	}
	delete(clients, client.Username)
	client.Conn.Close()
	fmt.Printf("INFO: %v: Client Disconnected\n", client.Username)
}

func handleMessages(messages <-chan Message) {
	for message := range messages {
		messageBytes := fmt.Appendf([]byte(""), "%s: %s", message.Author, message.Text)
		for _, client := range clients {
			if client.Username == message.Author {
				continue
			}

			select {
			case client.SendQueue <- messageBytes:
			default:
				handleClientDisconnect(client)
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
