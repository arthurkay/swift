package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CachedImage represents a cached backing image.
type CachedImage struct {
	URL       string
	LocalPath string
	SHA256    string
	Size      uint64
	CachedAt  int64
}

// ImageProgress is sent during image pull.
type ImageProgress struct {
	State          string
	BytesDownloaded uint64
	TotalBytes     uint64
	LocalPath      string
	Error          string
}

// ImageCache manages local caching of backing images.
type ImageCache struct {
	dir    string
	mu     sync.RWMutex
	images map[string]*CachedImage
}

// NewImageCache creates a new cache at the given directory.
func NewImageCache(dir string) *ImageCache {
	c := &ImageCache{
		dir:    dir,
		images: make(map[string]*CachedImage),
	}
	os.MkdirAll(dir, 0755)
	c.loadIndex()
	return c
}

// EnsureCached returns the local path of a cached image, pulling if needed.
func (c *ImageCache) EnsureCached(url, expectedSHA256 string) (string, error) {
	c.mu.RLock()
	img, ok := c.images[url]
	c.mu.RUnlock()

	if ok {
		// Verify file exists
		if _, err := os.Stat(img.LocalPath); err == nil {
			if expectedSHA256 == "" || img.SHA256 == expectedSHA256 {
				return img.LocalPath, nil
			}
		}
	}

	// Pull the image
	progressCh := c.Pull(url, expectedSHA256)
	for p := range progressCh {
		if p.Error != "" {
			return "", fmt.Errorf("pull image: %s", p.Error)
		}
		if p.State == "cached" {
			return p.LocalPath, nil
		}
	}
	return "", fmt.Errorf("image pull completed without path")
}

// Pull downloads an image from a URL, returning a channel of progress events.
func (c *ImageCache) Pull(url, expectedSHA256 string) <-chan *ImageProgress {
	ch := make(chan *ImageProgress, 10)

	go func() {
		defer close(ch)

		// Generate local filename from URL hash
		h := sha256.Sum256([]byte(url))
		filename := hex.EncodeToString(h[:8]) + ".qcow2"
		localPath := filepath.Join(c.dir, filename)

		// Check if already cached
		if info, err := os.Stat(localPath); err == nil {
			ch <- &ImageProgress{
				State:     "cached",
				LocalPath: localPath,
				TotalBytes: uint64(info.Size()),
			}
			return
		}

		// Start download
		ch <- &ImageProgress{State: "downloading"}

		resp, err := http.Get(url)
		if err != nil {
			ch <- &ImageProgress{Error: fmt.Sprintf("download: %v", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			ch <- &ImageProgress{Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
			return
		}

		totalBytes := uint64(resp.ContentLength)
		tmpPath := localPath + ".tmp"

		out, err := os.Create(tmpPath)
		if err != nil {
			ch <- &ImageProgress{Error: fmt.Sprintf("create file: %v", err)}
			return
		}
		defer func() {
			out.Close()
			os.Remove(tmpPath) // clean up on error
		}()

		hasher := sha256.New()
		writer := io.MultiWriter(out, hasher)

		var downloaded uint64
		buf := make([]byte, 32*1024)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				if _, wErr := writer.Write(buf[:n]); wErr != nil {
					ch <- &ImageProgress{Error: fmt.Sprintf("write: %v", wErr)}
					return
				}
				downloaded += uint64(n)
				ch <- &ImageProgress{
					State:          "downloading",
					BytesDownloaded: downloaded,
					TotalBytes:     totalBytes,
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				ch <- &ImageProgress{Error: fmt.Sprintf("read: %v", err)}
				return
			}
		}

		out.Close()

		// Verify SHA256
		actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
		if expectedSHA256 != "" && actualSHA256 != expectedSHA256 {
			os.Remove(tmpPath)
			ch <- &ImageProgress{
				Error: fmt.Sprintf("SHA256 mismatch: expected %s, got %s", expectedSHA256, actualSHA256),
			}
			return
		}

		// Move to final location
		if err := os.Rename(tmpPath, localPath); err != nil {
			ch <- &ImageProgress{Error: fmt.Sprintf("rename: %v", err)}
			return
		}

		// Index the cached image
		info, _ := os.Stat(localPath)
		cached := &CachedImage{
			URL:       url,
			LocalPath: localPath,
			SHA256:    actualSHA256,
			CachedAt:  time.Now().Unix(),
		}
		if info != nil {
			cached.Size = uint64(info.Size())
		}

		c.mu.Lock()
		c.images[url] = cached
		c.mu.Unlock()
		c.saveIndex()

		log.Printf("[cache] cached %s -> %s", url, localPath)

		ch <- &ImageProgress{
			State:     "cached",
			LocalPath: localPath,
		}
	}()

	return ch
}

// Status returns all cached images and total cache size.
func (c *ImageCache) Status() ([]*CachedImage, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalSize uint64
	var images []*CachedImage
	for _, img := range c.images {
		images = append(images, img)
		totalSize += img.Size
	}
	return images, totalSize
}

func (c *ImageCache) loadIndex() {
	// Simple JSON index file
	data, err := os.ReadFile(filepath.Join(c.dir, "index.json"))
	if err != nil {
		return
	}
	// Minimal implementation — would use JSON in production
	_ = data
}

func (c *ImageCache) saveIndex() {
	// Minimal implementation — would write JSON in production
}
