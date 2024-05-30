package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	serverPort          string
	backendAuthorities  string
	healthCheckInterval time.Duration

	roundRobinIdx   int
	healthyBackends []string
	backendsMutex   sync.RWMutex
)

func main() {
	flag.StringVar(&serverPort, "port", "8080", "server port")
	flag.StringVar(&backendAuthorities, "backends", "localhost:8081,localhost:8082", "comma-separated list of backend authorities")
	flag.DurationVar(&healthCheckInterval, "interval", 10*time.Second, "health check interval")
	flag.Parse()

	backends := strings.Split(backendAuthorities, ",")
	if len(backends) == 0 {
		log.Fatal("No backends provided")
	}

	updateHealthyBackends(backends)
	go startHealthCheck(backends, healthCheckInterval)

	http.HandleFunc("/", serve)
	log.Printf("Starting server on port :%s\n", serverPort)
	if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
		log.Fatalf("Error starting server: %s\n", err)
	}
}

func updateHealthyBackends(backends []string) {
	var newHealthyBackends []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, backend := range backends {
		wg.Add(1)
		go func(be string) {
			defer wg.Done()
			if isBackendHealthy(be) {
				mu.Lock()
				newHealthyBackends = append(newHealthyBackends, be)
				mu.Unlock()
			}
		}(backend)
	}

	wg.Wait()
	backendsMutex.Lock()
	healthyBackends = newHealthyBackends
	backendsMutex.Unlock()
	log.Printf("Healthy backends: %v\n", healthyBackends)
}

func startHealthCheck(backends []string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		updateHealthyBackends(backends)
	}
}

func isBackendHealthy(backend string) bool {
	url := "http://" + backend + "/health"
	start := time.Now()
	res, err := http.Get(url)
	duration := time.Since(start)
	if err != nil {
		log.Printf("Health check failed for %s: %s (took %s)\n", backend, err, duration)
		return false
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("Server %s is unhealthy: status code %d (took %s)\n", backend, res.StatusCode, duration)
		return false
	}

	log.Printf("Health check succeeded for %s (took %s)\n", backend, duration)
	return true
}

func serve(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	logRequest(r)

	backend, err := loadBalance()
	if err != nil {
		log.Printf("Error load balancing: %s\n", err)
		http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
		return
	}

	log.Printf("Forwarding request to backend: %s\n", backend)

	if err := handleRequest(w, backend); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error handling request: %s\n", err)
	}

	duration := time.Since(start)
	log.Printf("Request processed in %s\n", duration)
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

func loadBalance() (string, error) {
	backendsMutex.RLock()
	defer backendsMutex.RUnlock()

	if len(healthyBackends) == 0 {
		return "", errors.New("no healthy backends available")
	}

	if roundRobinIdx >= len(healthyBackends) {
		roundRobinIdx = 0
	}

	backend := healthyBackends[roundRobinIdx]
	roundRobinIdx++

	return backend, nil
}

func handleRequest(w http.ResponseWriter, backend string) error {
	start := time.Now()
	resBody, err := forwardRequest("http://" + backend)
	if err != nil {
		return fmt.Errorf("error making HTTP request: %s", err)
	}
	duration := time.Since(start)

	if _, err := w.Write(resBody); err != nil {
		return fmt.Errorf("error writing response: %s", err)
	}

	log.Printf("Request to backend %s took %s\n", backend, duration)
	return nil
}

func forwardRequest(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	log.Printf("Response from server: %s %s\n", res.Proto, res.Status)

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return resBody, nil
}
