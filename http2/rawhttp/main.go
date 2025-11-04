package main

import (
	"fmt"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func handlerFunc(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

func main() {
	h2s := &http2.Server{}
	handler := h2c.NewHandler(http.HandlerFunc(handlerFunc), h2s)

	server := &http.Server{
		Addr:    ":1010",
		Handler: handler,
	}

	fmt.Println("Starting server on :1010")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
