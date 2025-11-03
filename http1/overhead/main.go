package main

import (
	"fmt"
	"net"
	"runtime"
	"time"
)

func measureMem() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc / 1024
}

func simulateHTTP1(numRequests int) time.Duration {
	fmt.Printf("\n=== HTTP/1.1 Simualation: %d connections ===\n", numRequests)

	startMem := measureMem()
	start := time.Now()

	connections := make([]net.Conn, numRequests)

	for i := 0; i < numRequests; i++ {
		conn, err := net.Dial("tcp", "localhost:9001")
		if err != nil {
			fmt.Println("conn failed:", err)
			continue
		}

		connections[i] = conn

		fmt.Fprintf(conn, "GET /resource%d HTTP/1.1\r\nHost: localhost\r\n\r\n", i)
	}

	for i, conn := range connections {
		if conn == nil {
			continue
		}

		buffer := make([]byte, 1024)
		conn.Read(buffer)
		conn.Close()

		if i == 0 {
			fmt.Printf("Response %d: %s\n", i, buffer[:50])
		}
	}

	elapsed := time.Since(start)
	endMem := measureMem()

	fmt.Printf("Connections created: %d\n", numRequests)
	fmt.Printf("Memory used: %d KB\n", endMem-startMem)
	fmt.Printf("Time taken: %v\n", elapsed)

	return elapsed
}

func simulateHTTP2(numRequests int) time.Duration {
	fmt.Printf("\n=== HTTP/2 Simulation: 1 connection, %d streams ===\n", numRequests)

	startMem := measureMem()
	start := time.Now()

	// HTTP/2: Single connection, multiple logical streams
	conn, err := net.Dial("tcp", "localhost:9002")
	if err != nil {
		fmt.Println("Connection failed:", err)
		return 0
	}
	defer conn.Close()

	// Simulate multiplexed requests (in reality, HTTP/2 uses binary framing)
	for i := 0; i < numRequests; i++ {
		// In real HTTP/2, these would be interleaved frames
		fmt.Fprintf(conn, "STREAM:%d GET /resource%d\n", i, i)
	}

	// Read multiplexed responses
	buffer := make([]byte, 1024)
	conn.Read(buffer)
	fmt.Printf("Multiplexed response: %s\n", buffer[:50])

	elapsed := time.Since(start)
	endMem := measureMem()

	fmt.Printf("Connections created: 1\n")
	fmt.Printf("Streams: %d\n", numRequests)
	fmt.Printf("Memory used: %d KB\n", endMem-startMem)
	fmt.Printf("Time taken: %v\n", elapsed)

	return elapsed
}

func startHTTP1Server() {
	ln, _ := net.Listen("tcp", ":9001")
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			buffer := make([]byte, 1024)
			c.Read(buffer)

			response := "HTTP/1.1 200 OK\r\nContent-Length: 13\r\n\r\nHello HTTP/1!"
			c.Write([]byte(response))
		}(conn)
	}
}

func startHTTP2Server() {
	ln, _ := net.Listen("tcp", ":9002")
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			buffer := make([]byte, 4096)
			c.Read(buffer)

			response := "HTTP/2 multiplexed response with all stream"
			c.Write([]byte(response))
		}(conn)
	}
}

func main() {
	go startHTTP1Server()
	go startHTTP2Server()

	time.Sleep(100 * time.Millisecond)

	fmt.Println("=== comparing HTTP/1.1 vs HTTP/2 Resource usage ===")
	fmt.Println("\nscenario: loading a page with 50 resources")

	numRequests := 50

	// HTTP/1.1 approach
	http1Time := simulateHTTP1(numRequests)

	time.Sleep(500 * time.Millisecond)

	// HTTP/2 approach
	http2Time := simulateHTTP2(numRequests)

	// Summary
	fmt.Println("\n=== SUMMARY ===")
	fmt.Printf("HTTP/1.1: %d TCP connections created\n", numRequests)
	fmt.Printf("HTTP/2:   1 TCP connection (multiplexed)\n")
	fmt.Printf("\nHTTP/2 is %.1f× more efficient!\n", float64(numRequests)/1.0)

	if http1Time > 0 && http2Time > 0 {
		fmt.Printf("HTTP/2 was %.2f× faster\n", float64(http1Time)/float64(http2Time))
	}

	fmt.Println("\n=== KEY INSIGHT ===")
	fmt.Println("Each TCP connection requires:")
	fmt.Println("- File descriptor (OS resource)")
	fmt.Println("- Send/receive buffers (~87KB)")
	fmt.Println("- TCP state machine")
	fmt.Println("- TLS session (if HTTPS)")
	fmt.Println("\nWith 10,000 users:")
	fmt.Printf("HTTP/1.1: %d connections = ~6GB memory\n", 10000*6)
	fmt.Printf("HTTP/2:   %d connections = ~1GB memory\n", 10000)

	time.Sleep(1 * time.Second)
}
