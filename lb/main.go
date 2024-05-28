package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

var (
	serverPort         string
	backendAuthorities string
	roundRobinIdx      int
)

func main() {
	flag.StringVar(&serverPort, "port", "8080", "server port")
	flag.StringVar(&backendAuthorities, "backends", "localhost:8081,localhost:8082", "comma-separated list of backend authorities")
	flag.Parse()

	backends := parseAuthorities(backendAuthorities)
	if len(backends) == 0 {
		log.Fatal("No backends provided")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serve(w, r, backends)
	})

	log.Printf("Starting server on port :%s\n", serverPort)
	if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
		log.Fatalf("Error starting server: %s\n", err)
	}
}

func parseAuthorities(backendAuthorities string) []string {
	var backends []string
	for _, backend := range strings.Split(backendAuthorities, ",") {
		backends = append(backends, backend)
	}

	return backends
}

func serve(w http.ResponseWriter, r *http.Request, backends []string) {
	logRequest(r)

	var wg sync.WaitGroup
	wg.Add(1)

	backend := loadBalance(backends)
	go func(be string) {
		defer wg.Done()
		err := handleRequest(w, be)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error handling request: %s\n", err)
		}
	}(backend)

	wg.Wait()
}

func logRequest(r *http.Request) {
	log.Printf("Received request from %s\n", r.RemoteAddr)
	log.Printf("%s %s %s\n", r.Method, r.URL.Path, r.Proto)
	log.Printf("Host: %s\n", r.Host)
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("%s: %s\n", name, value)
		}
	}
}

func loadBalance(backends []string) string {
	if roundRobinIdx >= len(backends) {
		roundRobinIdx = 0
	}

	backend := backends[roundRobinIdx]
	roundRobinIdx++

	return backend
}

func handleRequest(w http.ResponseWriter, hostname string) error {
	resBody, err := doRequest(hostname)
	if err != nil {
		return fmt.Errorf("Error making http request: %s", err)
	}

	if _, err := w.Write(resBody); err != nil {
		return fmt.Errorf("Error writing response: %s", err)
	}

	return nil
}

func doRequest(backend string) ([]byte, error) {
	res, err := http.Get("http://" + backend)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	log.Printf("Response from server: %s %d %s\n", res.Proto, res.StatusCode, res.Status)

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return resBody, nil
}
