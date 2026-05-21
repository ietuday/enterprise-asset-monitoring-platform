package prometheus

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func Reload() error {
	reloadURL := os.Getenv("PROMETHEUS_RELOAD_URL")
	if reloadURL == "" {
		reloadURL = "http://localhost:9090/-/reload"
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(http.MethodPost, reloadURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reload prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("prometheus reload failed with status %d", resp.StatusCode)
	}

	return nil
}