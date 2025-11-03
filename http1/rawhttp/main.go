package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		requestLine, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("client closed connection")
			} else {
				fmt.Println("error reading request: ", err)
			}
			return
		}

		fmt.Println("received request: ", strings.TrimSpace(requestLine))

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if line == "\r\n" {
				break
			}
		}

		response := "HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/html\r\n" +
			"Content-Length: 37\r\n" +
			"Connection: Keep-Alive\r\n" +
			"Keep-Alive: timeout=5, max=100\r\n" +
			"\r\n" +
			"<html><body><h1>OK</h1></body></html>"

		_, err = fmt.Fprint(conn, response)
		if err != nil {
			fmt.Println("error writing response: ", err)
			return
		}

		fmt.Println("sent response")
	}
}

func main() {
	go func() {
		time.Sleep(100 * time.Millisecond)

		conn, err := net.Dial("tcp", "localhost:8080")
		if err != nil {
			panic(err)
		}
		defer conn.Close()

		fmt.Println("\n==== CLIENT: Sending first request ====")
		fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: localhost:8080\r\n\r\n")

		reader := bufio.NewReader(conn)
		fmt.Println("first response:")
		readResponse(reader)

		fmt.Println("\n==== CLIENT: Sending second request ====")
		time.Sleep(500 * time.Millisecond)
		fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: localhost:8080\r\n\r\n")

		fmt.Println("second response:")
		readResponse(reader)

		fmt.Println("\n==== CLIENT: Sending third request ====")
		time.Sleep(500 * time.Millisecond)
		fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: localhost:8080\r\n\r\n")

		fmt.Println("third response:")
		readResponse(reader)

		fmt.Println("\n==== CLIENT: Completed ====")
	}()

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go handleConnection(conn)
	}
}

func readResponse(reader *bufio.Reader) {
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("error reading status line: ", err)
		return
	}

	fmt.Println(strings.TrimSpace(statusLine))

	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}

		fmt.Println(trimmed)

		if strings.HasPrefix(trimmed, "Content-Length:") {
			fmt.Sscanf(trimmed, "Content-Length: %d", &contentLength)
		}
	}

	if contentLength > 0 {
		body := make([]byte, contentLength)
		io.ReadFull(reader, body)
		fmt.Println(string(body))
	}
}
