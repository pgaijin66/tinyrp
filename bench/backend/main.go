package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := "9090"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	body := []byte(`{"status":"ok","data":"benchmark payload for proxy comparison"}`)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	})
	fmt.Printf("backend listening on :%s\n", port)
	http.ListenAndServe(":"+port, nil)
}
