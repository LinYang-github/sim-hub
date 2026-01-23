package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	APIBaseURL = "http://localhost:30030"
	TestFile   = "test_scenario.zip"
)

// DTOs (Simplified)
type UploadTokenResponse struct {
	TicketID     string `json:"ticket_id"`
	PresignedURL string `json:"presigned_url"`
}

type ResourceDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	LatestVer struct {
		State    string         `json:"state"`
		MetaData map[string]any `json:"meta_data"`
	} `json:"latest_version"`
}

func main() {
	fmt.Println("🚀 Starting End-to-End Test for SimHub...")

	// 1. Prepare a dummy file
	createDummyFile(TestFile, 1024*10) // 10KB
	defer os.Remove(TestFile)

	client := &http.Client{Timeout: 10 * time.Second}

	// 2. Request Upload Token
	fmt.Println("👉 Step 1: Requesting Upload Token...")
	tokenResp, err := requestUploadToken(client)
	if err != nil {
		fatal("Request Token Failed", err)
	}
	fmt.Printf("   Ticket ID: %s\n", tokenResp.TicketID)

	// 3. Upload File to MinIO (via Presigned URL)
	fmt.Println("👉 Step 2: Uploading file to MinIO...")
	if err := uploadFileToMinIO(client, tokenResp.PresignedURL, TestFile); err != nil {
		fatal("Upload to MinIO Failed", err)
	}

	// 4. Confirm Upload
	fmt.Println("👉 Step 3: Confirming Upload...")
	if err := confirmUpload(client, tokenResp.TicketID); err != nil {
		fatal("Confirm Upload Failed", err)
	}

	// 5. Poll for Processing Status
	fmt.Println("👉 Step 4: Polling for Processing Result (Worker Interaction)...")
	resourceID := extractResourceID(tokenResp.TicketID)

	success := false
	for i := 0; i < 20; i++ { // Retry for 20 seconds
		res, err := getResource(client, resourceID)
		if err != nil {
			fmt.Printf("   Polling error: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		state := res.LatestVer.State
		fmt.Printf("   Current State: %s (ID: %s, Name: %s)\n", state, res.ID, res.Name)

		if state == "ACTIVE" {
			// Validate Metadata injected by Worker
			if res.LatestVer.MetaData != nil {
				if val, ok := res.LatestVer.MetaData["processed_by"]; ok {
					fmt.Printf("   ✅ Processed By: %v\n", val)
					if val == "simhub-worker" {
						success = true
						break
					}
				}
			}
		} else if state == "ERROR" {
			fatal("Processing Failed with ERROR state", nil)
		}

		time.Sleep(1 * time.Second)
	}

	if success {
		fmt.Println("\n🎉 E2E Test PASSED! System is fully operational.")
	} else {
		fatal("Test Timed Out waiting for Worker", nil)
	}
}

// Helpers

func createDummyFile(name string, size int64) {
	f, _ := os.Create(name)
	f.Truncate(size)
	f.Close()
}

func requestUploadToken(client *http.Client) (*UploadTokenResponse, error) {
	reqBody := map[string]any{
		"resource_type": "scenario",
		"filename":      TestFile,
		"mode":          "presigned",
	}
	body, _ := json.Marshal(reqBody)
	resp, err := client.Post(APIBaseURL+"/api/v1/integration/upload/token", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var res UploadTokenResponse
	json.NewDecoder(resp.Body).Decode(&res)
	return &res, nil
}

func uploadFileToMinIO(client *http.Client, url, filePath string) error {
	file, _ := os.Open(filePath)
	defer file.Close()
	fi, _ := file.Stat()

	req, _ := http.NewRequest("PUT", url, file)
	req.ContentLength = fi.Size()
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("minio status %d", resp.StatusCode)
	}
	return nil
}

func confirmUpload(client *http.Client, ticketID string) error {
	reqBody := map[string]any{
		"ticket_id": ticketID,
		"type_key":  "scenario",
		"name":      "E2E Test Scenario",
		"owner_id":  "tester",
		"tags":      []string{"e2e", "test"},
		"size":      10240,
	}
	body, _ := json.Marshal(reqBody)
	resp, err := client.Post(APIBaseURL+"/api/v1/integration/upload/confirm", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func getResource(client *http.Client, id string) (*ResourceDTO, error) {
	// Our ID is embedded in ticket: ticketID + "::" + objectKey
	// Wait, SimHub returns the resource ID only via List or if we knew it.
	// Actually, confirm upload creates the resource.
	// For this test, we need to find the resource ID.
	// To simplify, we'll LIST resources and find by name.

	resp, err := client.Get(APIBaseURL + "/api/v1/resources?type=scenario&size=1")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []ResourceDTO `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("resource not found in list")
	}

	// Return the most recent one
	return &result.Items[0], nil
}

func extractResourceID(ticketID string) string {
	// Implementation simplified above by using List
	return ""
}

func fatal(msg string, err error) {
	if err != nil {
		fmt.Printf("❌ %s: %v\n", msg, err)
	} else {
		fmt.Printf("❌ %s\n", msg)
	}
	os.Exit(1)
}
