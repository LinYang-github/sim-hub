package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	simhub "io.simhub/sdk/go"
)

func main() {
	baseURL := "http://localhost:30030"
	token := "shp_1b97e9599aa5e36917afc106493338f137bef60acd54b60dacef41823e574c99"

	client := simhub.NewClient(baseURL, token)
	client.SetConcurrency(8)

	ctx := context.Background()

	// 1. List Resources
	fmt.Println(">>> Listing resources...")
	res, err := client.ListResources(ctx, "scenario", 1, 10)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total resources: %d\n", res.Total)
	for _, item := range res.Items {
		fmt.Printf("- [%s] %s\n", item.ID, item.Name)
	}

	// 2. Upload Example
	fmt.Println("\n>>> Preparing a dummy file for upload...")
	filename := "go_test_file.txt"
	content := []byte("Hello SimHub from Go SDK! " + time.Now().Format(time.RFC3339))
	err = os.WriteFile(filename, content, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(filename)

	fmt.Println(">>> Uploading file...")
	err = client.UploadFileMultipart(ctx, "documents", filename, "Go SDK Upload Test", "1.0.0", func(uploaded, total int64) {
		fmt.Printf("\rProgress: %.2f%%", float64(uploaded)/float64(total)*100)
	})
	if err != nil {
		log.Fatalf("\nUpload error: %v", err)
	}
	fmt.Println("\nUpload successful!")

	// 3. Download the first resource
	if len(res.Items) > 0 {
		firstID := res.Items[0].ID
		fmt.Printf("\n>>> Downloading resource %s...\n", firstID)
		err = client.DownloadFile(ctx, firstID, "downloaded_"+firstID+".zip", nil)
		if err != nil {
			fmt.Printf("Download error (expected if URL is expired): %v\n", err)
		} else {
			fmt.Println("Download successful!")
			os.Remove("downloaded_" + firstID + ".zip")
		}
	}
}
