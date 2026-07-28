package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := flag.String("addr", ":8080", "Address to listen on.")
	root := flag.String("root", ".", "Directory to serve.")
	timeout := flag.Duration("timeout", 10*time.Second, "Deadline for reading a request.")
	flag.Parse()

	server := &http.Server{
		Addr:         *addr,
		Handler:      http.FileServer(http.Dir(*root)),
		ReadTimeout:  *timeout,
		WriteTimeout: *timeout,
	}

	fmt.Fprintf(os.Stderr, "serving %s on %s\n", *root, *addr)
	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, "serve:", err)
		os.Exit(1)
	}
}
