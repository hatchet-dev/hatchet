package whrequest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hatchet-dev/hatchet/internal/signature"
)

func Send(ctx context.Context, url string, secret string, data any, headers ...func(req *http.Request)) ([]byte, *int, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}

	sig, err := signature.Sign(string(body), secret)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("X-Hatchet-Signature", sig)
	req.Header.Set("Content-Type", "application/json")

	for _, h := range headers {
		h(req)
	}

	httpClient := &http.Client{
		// use 10 minutes timeout
		Timeout: time.Second * 600,
	}

	// TODO block-list

	// nolint:gosec
	resp, err := httpClient.Do(req)
	if err != nil {
		connRefused := 502
		return nil, &connRefused, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &resp.StatusCode, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &resp.StatusCode, fmt.Errorf("could not read response body: %w", err)
	}

	return res, &resp.StatusCode, nil
}
