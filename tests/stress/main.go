package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

var (
	baseURL     = flag.String("url", "http://localhost:30030", "SimHub API Base URL")
	concurrency = flag.Int("c", 10, "Number of concurrent workers")
	duration    = flag.Duration("d", 10*time.Second, "Test duration")
)

type Result struct {
	Latency time.Duration
	Err     error
}

func main() {
	flag.Parse()

	fmt.Printf("Starting stress test against %s\n", *baseURL)
	fmt.Printf("Concurrency: %d, Duration: %v\n", *concurrency, *duration)

	results := make(chan Result, 10000)
	var wg sync.WaitGroup

	stop := make(chan struct{})
	timer := time.NewTimer(*duration)

	// Launch workers
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					start := time.Now()
					err := performWorkflow(*baseURL)
					if err != nil {
						fmt.Printf("Sample Error: %v\n", err)
						return // Stop this worker on error to avoid flooding
					}
					results <- Result{Latency: time.Since(start), Err: err}
				}
			}
		}(i)
	}

	go func() {
		<-timer.C
		close(stop)
	}()

	// Collector
	done := make(chan struct{})
	var totalReq, successReq int
	var totalLatency time.Duration

	go func() {
		for res := range results {
			totalReq++
			if res.Err == nil {
				successReq++
				totalLatency += res.Latency
			}
		}
		close(done)
	}()

	wg.Wait()
	close(results)
	<-done

	fmt.Println("\n--- Stress Test Result ---")
	fmt.Printf("Total Requests:   %d\n", totalReq)
	fmt.Printf("Successful:      %d\n", successReq)
	fmt.Printf("Failed:          %d\n", totalReq-successReq)
	if successReq > 0 {
		fmt.Printf("Avg Latency:     %v\n", totalLatency/time.Duration(successReq))
		fmt.Printf("Requests/sec:    %.2f\n", float64(totalReq)/duration.Seconds())
	}
}

func performWorkflow(base string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Apply Token
	tokenReq := map[string]interface{}{
		"resource_type": "scenario",
		"filename":      "stress_test.bin",
		"mode":          "presigned",
	}
	body, _ := json.Marshal(tokenReq)
	resp, err := client.Post(base+"/api/v1/integration/upload/token", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token status: %d", resp.StatusCode)
	}

	var tokenRes struct {
		TicketID     string `json:"ticket_id"`
		PresignedURL string `json:"presigned_url"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenRes)

	// 2. Simple List (Simulating heavy read)
	respList, err := client.Get(base + "/api/v1/resources?type=scenario")
	if err != nil {
		return err
	}
	defer respList.Body.Close()
	io.Copy(io.Discard, respList.Body)

	return nil
}
