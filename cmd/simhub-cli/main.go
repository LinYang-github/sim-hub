package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Config
const serverURL = "http://localhost:8080"

func main() {
	filePtr := flag.String("file", "", "Path to file to upload")
	typePtr := flag.String("type", "map_terrain", "Resource Type Key")
	flag.Parse()

	if *filePtr == "" {
		log.Fatal("Please provide a file using -file")
	}

	filePath := *filePtr
	filename := filepath.Base(filePath)
	fmt.Printf("1. Starting upload process for %s (Type: %s)\n", filename, *typePtr)

	// Step 1: Request Token
	fmt.Println("\n[Step 1] Requesting Upload Token from SimHub...")
	tokenResp, err := requestToken(filename, *typePtr)
	if err != nil {
		log.Fatalf("Request Token Failed: %v", err)
	}
	fmt.Printf("   -> Ticket ID: %s\n", tokenResp.TicketID)
	fmt.Printf("   -> Presigned URL: %s...\n", tokenResp.PresignedURL[:50])

	// Step 2: Upload to MinIO
	fmt.Println("\n[Step 2] Uploading file to MinIO...")
	if err := uploadFile(filePath, tokenResp.PresignedURL); err != nil {
		log.Fatalf("Upload Failed: %v", err)
	}
	fmt.Println("   -> Upload Successful")

	// Step 3: Confirm Upload
	fmt.Println("\n[Step 3] Confirming Upload to SimHub...")
	fileInfo, _ := os.Stat(filePath)
	if err := confirmUpload(tokenResp.TicketID, filename, *typePtr, fileInfo.Size()); err != nil {
		log.Fatalf("Confirm Failed: %v", err)
	}
	fmt.Println("   -> Confirmation Successful! Resource created.")
}

type TokenResponse struct {
	TicketID     string `json:"ticket_id"`
	PresignedURL string `json:"presigned_url"`
	Bucket       string `json:"bucket"`
	Prefix       string `json:"prefix"`
}

func requestToken(filename, typeKey string) (*TokenResponse, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"filename": filename,
		"type":     typeKey,
	})
	resp, err := http.Post(serverURL+"/api/v1/integration/upload/token", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func uploadFile(path, url string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, _ := file.Stat()

	req, err := http.NewRequest("PUT", url, file)
	if err != nil {
		return err
	}
	req.ContentLength = stat.Size()
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func confirmUpload(ticketID, filename, typeKey string, size int64) error {
	reqBody, _ := json.Marshal(map[string]any{
		"ticket_id":  ticketID,
		"filename":   filename,
		"type_key":   typeKey,
		"name":       "CLI Uploaded: " + filename,
		"owner_id":   "admin_cli",
		"size":       size,
		"extra_meta": map[string]any{"source": "cli"},
	})
	resp, err := http.Post(serverURL+"/api/v1/integration/upload/confirm", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
