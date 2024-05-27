package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
)

var (
	serverPort   string
	backendPorts string
)

func main() {
	flag.StringVar(&serverPort, "port", "8080", "server port")
	flag.StringVar(&backendPorts, "backends", "8081,8082", "comma-separated list of backend ports")
	flag.Parse()

	ports := parseBackendPorts(backendPorts)
	if len(ports) == 0 {
		log.Fatal("No backend ports provided")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serve(w, r, ports)
	})

	log.Printf("Starting server on port :%s\n", serverPort)
	if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
		log.Fatalf("Error starting server: %s\n", err)
	}
}

func parseBackendPorts(backendPorts string) []int {
	var ports []int
	for _, portStr := range splitPorts(backendPorts) {
		var port int
		_, err := fmt.Sscanf(portStr, "%d", &port)
		if err == nil {
			ports = append(ports, port)
		}
	}
	return ports
}

func splitPorts(backendPorts string) []string {
	return strings.Split(backendPorts, ",")
}

func serve(w http.ResponseWriter, r *http.Request, backendPorts []int) {
	logRequest(r)

	var wg sync.WaitGroup
	wg.Add(1)

	backendPort := loadBalance(backendPorts)
	go func(port int) {
		defer wg.Done()
		err := handleRequest(w, port)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error handling request: %s\n", err)
		}
	}(backendPort)

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

func loadBalance(backendPorts []int) int {
	return backendPorts[rand.Intn(len(backendPorts))]
}

func handleRequest(w http.ResponseWriter, backendPort int) error {
	resBody, err := doRequest(backendPort)
	if err != nil {
		return fmt.Errorf("Error making http request: %s", err)
	}

	if _, err := w.Write(resBody); err != nil {
		return fmt.Errorf("Error writing response: %s", err)
	}

	return nil
}

func doRequest(backendPort int) ([]byte, error) {
	reqURL := fmt.Sprintf("http://localhost:%d", backendPort)
	res, err := http.Get(reqURL)
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
