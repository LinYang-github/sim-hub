package simhub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type Client struct {
	BaseURL     string
	Token       string
	HTTPClient  *http.Client
	Concurrency int
}

type ProgressFunc func(uploaded, total int64)

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:     baseURL,
		Token:       token,
		HTTPClient:  &http.Client{},
		Concurrency: 4,
	}
}

func (c *Client) SetConcurrency(n int) {
	c.Concurrency = n
}

func (c *Client) request(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}

	return data, nil
}

func (c *Client) ListResourceTypes(ctx context.Context) ([]ResourceType, error) {
	data, err := c.request(ctx, "GET", "/api/v1/resource-types", nil)
	if err != nil {
		return nil, err
	}

	var res []ResourceType
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Client) ListResources(ctx context.Context, typeKey string, page, size int) (*ResourceListResponse, error) {
	path := fmt.Sprintf("/api/v1/resources?type_key=%s&page=%d&size=%d", url.QueryEscape(typeKey), page, size)
	data, err := c.request(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var res ResourceListResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) GetResource(ctx context.Context, id string) (*Resource, error) {
	data, err := c.request(ctx, "GET", "/api/v1/resources/"+id, nil)
	if err != nil {
		return nil, err
	}

	var res Resource
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) UploadFileSimple(ctx context.Context, typeKey, filePath, name, semver string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	// 1. Get Token
	tokenReq := map[string]any{
		"resource_type": typeKey,
		"filename":      filepath.Base(filePath),
		"size":          fi.Size(),
	}
	data, err := c.request(ctx, "POST", "/api/v1/integration/upload/token", tokenReq)
	if err != nil {
		return err
	}

	var tokenResp UploadTokenResponse
	if err := json.Unmarshal(data, &tokenResp); err != nil {
		return err
	}

	// 2. Upload to S3
	req, err := http.NewRequestWithContext(ctx, "PUT", tokenResp.PresignedURL, file)
	if err != nil {
		return err
	}
	req.ContentLength = fi.Size()

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload failed: status %d", resp.StatusCode)
	}

	// 3. Confirm
	confirmReq := map[string]any{
		"ticket_id": tokenResp.TicketID,
		"name":      name,
		"semver":    semver,
	}
	_, err = c.request(ctx, "POST", "/api/v1/integration/upload/confirm", confirmReq)
	return err
}

func (c *Client) UploadFileMultipart(ctx context.Context, typeKey, filePath, name, semver string, progress ProgressFunc) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}
	totalSize := fi.Size()
	partSize := int64(5 * 1024 * 1024)
	partCount := int((totalSize + partSize - 1) / partSize)

	// 1. Init
	initReq := map[string]any{
		"resource_type": typeKey,
		"filename":      filepath.Base(filePath),
		"part_count":    partCount,
	}
	data, err := c.request(ctx, "POST", "/api/v1/integration/upload/multipart/init", initReq)
	if err != nil {
		return err
	}

	var initResp struct {
		TicketID  string `json:"ticket_id"`
		UploadID  string `json:"upload_id"`
		ObjectKey string `json:"object_key"`
	}
	if err := json.Unmarshal(data, &initResp); err != nil {
		return err
	}

	// 2. Parallel Upload
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, c.Concurrency)
	etags := make([]PartETag, partCount)
	errChan := make(chan error, partCount)
	var uploaded int64

	for i := 1; i <= partCount; i++ {
		wg.Add(1)
		go func(partNum int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			offset := int64(partNum-1) * partSize
			size := partSize
			if offset+size > totalSize {
				size = totalSize - offset
			}

			// Read part data
			partData := make([]byte, size)
			_, err := file.ReadAt(partData, offset)
			if err != nil {
				errChan <- err
				return
			}

			// Get Part URL
			partPath := "/api/v1/integration/upload/multipart/part-url"
			payload := map[string]any{
				"upload_id":   initResp.UploadID,
				"ticket_id":   initResp.TicketID,
				"part_number": partNum,
			}
			pdata, err := c.request(ctx, "POST", partPath, payload)
			if err != nil {
				errChan <- err
				return
			}
			var urlMap map[string]string
			json.Unmarshal(pdata, &urlMap)
			presignedURL := urlMap["url"]

			// Upload Part
			req, err := http.NewRequestWithContext(ctx, "PUT", presignedURL, bytes.NewReader(partData))
			if err != nil {
				errChan <- err
				return
			}
			resp, err := c.HTTPClient.Do(req)
			if err != nil {
				errChan <- err
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				errChan <- fmt.Errorf("part %d failed: status %d", partNum, resp.StatusCode)
				return
			}

			etag := resp.Header.Get("ETag")
			if etag != "" && etag[0] == '"' {
				etag = etag[1 : len(etag)-1]
			}
			etags[partNum-1] = PartETag{PartNumber: partNum, ETag: etag}

			currentUploaded := atomic.AddInt64(&uploaded, size)
			if progress != nil {
				progress(currentUploaded, totalSize)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)
	if err := <-errChan; err != nil {
		return err
	}

	// 3. Complete
	completeReq := map[string]any{
		"upload_id": initResp.UploadID,
		"ticket_id": initResp.TicketID,
		"parts":     etags,
		"type_key":  typeKey,
		"name":      name,
		"semver":    semver,
		"owner_id":  "admin",
		"scope":     "public",
	}
	_, err = c.request(ctx, "POST", "/api/v1/integration/upload/multipart/complete", completeReq)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) DownloadFile(ctx context.Context, id, targetPath string, progress ProgressFunc) error {
	res, err := c.GetResource(ctx, id)
	if err != nil {
		return err
	}

	if res.LatestVersion.DownloadURL == "" {
		return fmt.Errorf("resource has no download URL")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", res.LatestVersion.DownloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if progress != nil {
		total := resp.ContentLength
		var current int64
		buffer := make([]byte, 32*1024)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				if _, errW := out.Write(buffer[:n]); errW != nil {
					return errW
				}
				current += int64(n)
				progress(current, total)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
		}
		return nil
	}

	_, err = io.Copy(out, resp.Body)
	return err
}
