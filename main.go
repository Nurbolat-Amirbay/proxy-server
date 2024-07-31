package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
	"github.com/google/uuid"
)

type Request struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type Response struct {
	ID      string              `json:"id"`
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Length  int                 `json:"length"`
}

var requestStore sync.Map

func main() {
	http.HandleFunc("/proxy", handleProxy)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Method == "" || req.URL == "" {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	requestID := uuid.New().String()
	requestStore.Store(requestID, req)

	client := &http.Client{Timeout: 10 * time.Second}
	reqOut, err := http.NewRequest(req.Method, req.URL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	for k, v := range req.Headers {
		reqOut.Header.Add(k, v)
	}

	resp, err := client.Do(reqOut)
	if err != nil {
		http.Error(w, "Failed to perform request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	response := Response{
		ID:      requestID,
		Status:  resp.StatusCode,
		Headers: resp.Header,
		Length:  len(respBody),
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}
