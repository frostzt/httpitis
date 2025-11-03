package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// HTTP/1.1 Server
func handleHTTP1(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for i := 0; i < 3; i++ {
		// Read request line
		requestLine, _ := reader.ReadString('\n')
		fmt.Printf("SERVER got: %s", requestLine)

		// Read headers until empty line
		for {
			line, _ := reader.ReadString('\n')
			if line == "\r\n" {
				break
			}
		}

		// Simulate slow response (like downloading a large file)
		if i == 0 {
			fmt.Println("SERVER: Sending SLOW response (3 seconds)...")
			time.Sleep(3 * time.Second)
		}

		response := fmt.Sprintf("HTTP/1.1 200 OK\r\n"+
			"Content-Length: 20\r\n"+
			"\r\n"+
			"Response #%d content!", i+1)

		conn.Write([]byte(response))
		fmt.Printf("SERVER: Sent response #%d\n\n", i+1)
	}
}

func testHTTP1Pipelining() {
	fmt.Println("=== HTTP/1.1 'PIPELINING' ATTEMPT ===\n")

	conn, _ := net.Dial("tcp", "localhost:8001")
	defer conn.Close()

	fmt.Println("CLIENT: Sending 3 requests immediately (without waiting)...\n")

	// Send all 3 requests at once (pipelining)
	start := time.Now()
	conn.Write([]byte("GET /resource1 HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	conn.Write([]byte("GET /resource2 HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	conn.Write([]byte("GET /resource3 HTTP/1.1\r\nHost: localhost\r\n\r\n"))

	fmt.Println("CLIENT: All requests sent! Now reading responses...\n")

	reader := bufio.NewReader(conn)

	// Try to read responses
	for i := 0; i < 3; i++ {
		fmt.Printf("CLIENT: Waiting for response #%d...\n", i+1)

		// Read status line
		statusLine, _ := reader.ReadString('\n')
		fmt.Printf("CLIENT got: %s", statusLine)

		// Read headers
		contentLength := 0
		for {
			line, _ := reader.ReadString('\n')
			if line == "\r\n" {
				break
			}
			if strings.HasPrefix(line, "Content-Length:") {
				fmt.Sscanf(line, "Content-Length: %d", &contentLength)
			}
		}

		// Read body
		body := make([]byte, contentLength)
		io.ReadFull(reader, body)
		fmt.Printf("CLIENT received: %s\n", body)
		fmt.Printf("Time elapsed: %v\n\n", time.Since(start))
	}

	elapsed := time.Since(start)
	fmt.Printf("=== TOTAL TIME: %v ===\n", elapsed)
	fmt.Println("\n❌ PROBLEM: Response #1 took 3 seconds.")
	fmt.Println("   Responses #2 and #3 had to WAIT even though they were ready!")
	fmt.Println("   This is HEAD-OF-LINE BLOCKING!\n")
}

// HTTP/2 Server (simulated with multiplexing support)
func handleHTTP2(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// Read all requests first
	requests := make([]string, 0)
	for i := 0; i < 3; i++ {
		line, _ := reader.ReadString('\n')
		requests = append(requests, line)
		fmt.Printf("SERVER got stream #%d: %s", i, line)
	}

	fmt.Println("\nSERVER: Processing all streams concurrently...\n")

	// Simulate concurrent processing with channels
	responses := make(chan string, 3)

	for i, req := range requests {
		go func(streamID int, request string) {
			if streamID == 0 {
				fmt.Printf("SERVER: Stream #%d is slow (3 seconds)...\n", streamID)
				time.Sleep(3 * time.Second)
			} else {
				fmt.Printf("SERVER: Stream #%d is fast (0.1 seconds)...\n", streamID)
				time.Sleep(100 * time.Millisecond)
			}

			response := fmt.Sprintf("STREAM:%d Response #%d ready!\n", streamID, streamID+1)
			responses <- response
		}(i, req)
	}

	// Send responses as they become ready (interleaved!)
	for i := 0; i < 3; i++ {
		resp := <-responses
		conn.Write([]byte(resp))
		fmt.Printf("SERVER: Sent %s", resp)
	}
}

func testHTTP2Multiplexing() {
	fmt.Println("\n\n=== HTTP/2 MULTIPLEXING ===\n")

	conn, _ := net.Dial("tcp", "localhost:8002")
	defer conn.Close()

	fmt.Println("CLIENT: Sending 3 stream requests...\n")

	start := time.Now()
	// In real HTTP/2, these are binary frames with stream IDs
	conn.Write([]byte("STREAM:0 GET /resource1\n"))
	conn.Write([]byte("STREAM:1 GET /resource2\n"))
	conn.Write([]byte("STREAM:2 GET /resource3\n"))

	fmt.Println("CLIENT: Receiving multiplexed responses...\n")

	reader := bufio.NewReader(conn)
	responseTimes := make(map[int]time.Duration)

	for i := 0; i < 3; i++ {
		line, _ := reader.ReadString('\n')
		elapsed := time.Since(start)

		var streamID int
		fmt.Sscanf(line, "STREAM:%d", &streamID)
		responseTimes[streamID] = elapsed

		fmt.Printf("CLIENT received (%.2fs): %s", elapsed.Seconds(), line)
	}

	elapsed := time.Since(start)
	fmt.Printf("\n=== TOTAL TIME: %v ===\n", elapsed)
	fmt.Println("\n✅ SUCCESS: Stream #1 and #2 didn't wait for slow Stream #0!")
	fmt.Printf("   Stream #1 arrived at: %.2fs\n", responseTimes[1].Seconds())
	fmt.Printf("   Stream #2 arrived at: %.2fs\n", responseTimes[2].Seconds())
	fmt.Printf("   Stream #0 arrived at: %.2fs\n", responseTimes[0].Seconds())
	fmt.Println("   Responses arrived OUT OF ORDER - that's multiplexing!\n")
}

func main() {
	// Start HTTP/1.1 server
	go func() {
		ln, _ := net.Listen("tcp", ":8001")
		for {
			conn, _ := ln.Accept()
			go handleHTTP1(conn)
		}
	}()

	// Start HTTP/2 server
	go func() {
		ln, _ := net.Listen("tcp", ":8002")
		for {
			conn, _ := ln.Accept()
			go handleHTTP2(conn)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  Demonstrating HTTP/1.1 vs HTTP/2 Multiplexing        ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝\n")

	// Test HTTP/1.1
	testHTTP1Pipelining()

	time.Sleep(1 * time.Second)

	// Test HTTP/2
	testHTTP2Multiplexing()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("KEY DIFFERENCE:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nHTTP/1.1:")
	fmt.Println("  - Requests sent together (pipelining)")
	fmt.Println("  - Responses MUST arrive in order")
	fmt.Println("  - Response #2 blocked by slow Response #1")
	fmt.Println("  - Total time: ~3 seconds (serial)")
	fmt.Println("\nHTTP/2:")
	fmt.Println("  - Requests sent together (streams)")
	fmt.Println("  - Responses can arrive in ANY order")
	fmt.Println("  - Fast responses don't wait for slow ones")
	fmt.Println("  - Total time: ~3 seconds but fast responses arrive early!")
	fmt.Println(strings.Repeat("=", 60))

	time.Sleep(2 * time.Second)
}
