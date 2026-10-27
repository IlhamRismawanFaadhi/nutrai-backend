// Package n8n berisi HTTP client untuk memanggil webhook n8n (sync, dengan retry).
package n8n

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"
)

type Client struct {
	http    *http.Client
	secret  string
	retries int
}

// Error dikembalikan ketika n8n membalas HTTP status >= 400.
type Error struct {
	Status int
	Body   string
}

func (e *Error) Error() string { return fmt.Sprintf("n8n returned HTTP %d: %s", e.Status, e.Body) }

// NewClient: timeout berlaku PER percobaan.
func NewClient(secret string, timeout time.Duration, retries int) *Client {
	return &Client{http: &http.Client{Timeout: timeout}, secret: secret, retries: retries}
}

// PostJSON mengirim payload JSON dan men-decode respons ke out.
func (c *Client) PostJSON(ctx context.Context, url string, payload, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	return c.do(ctx, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}, out)
}

// PostMultipart mengirim 1 file + field teks (binary di n8n tersedia sebagai property `fileField`).
func (c *Client) PostMultipart(ctx context.Context, url string, fields map[string]string,
	fileField, filename, contentType string, data []byte, out any) error {

	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return err
		}
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, fileField, filename))
	h.Set("Content-Type", contentType)
	part, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	if _, err := part.Write(data); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	raw, ctype := buf.Bytes(), mw.FormDataContentType()

	return c.do(ctx, func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", ctype)
		return req, nil
	}, out)
}

// do menjalankan request dengan retry (error jaringan & 5xx). 4xx tidak di-retry.
func (c *Client) do(ctx context.Context, build func() (*http.Request, error), out any) error {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
		req, err := build()
		if err != nil {
			return err
		}
		req = req.WithContext(ctx)
		req.Header.Set("Accept", "application/json")
		if c.secret != "" {
			req.Header.Set("X-N8N-SECRET", c.secret)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil { // client batal / deadline request
				return ctx.Err()
			}
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()

		switch {
		case resp.StatusCode >= 500:
			lastErr = &Error{Status: resp.StatusCode, Body: truncate(body)}
			continue
		case resp.StatusCode >= 400:
			return &Error{Status: resp.StatusCode, Body: truncate(body)}
		}
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode n8n response: %w (body: %s)", err, truncate(body))
		}
		return nil
	}
	return fmt.Errorf("n8n request failed after %d attempt(s): %w", c.retries+1, lastErr)
}

func truncate(b []byte) string {
	if len(b) > 300 {
		return string(b[:300]) + "..."
	}
	return string(b)
}
