package main

import (
	"flag"
	"log"
	"net/http"
)

var (
	serverPort string
)

func main() {
	flag.StringVar(&serverPort, "port", "8080", "server port")
	flag.Parse()

	http.HandleFunc("/", serve)

	log.Printf("Starting server on port :%s\n", serverPort)
	if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
		log.Fatalf("Error starting server: %s\n", err)
	}
}

func serve(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	resBody := "Hello from backend server"
	log.Printf("Response: %s\n", resBody)
	w.Write([]byte(resBody))
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
