package mediamtx

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"mio9/mtx-monitor/internal/config"
	"mio9/mtx-monitor/internal/constants"
)

// Client communicates with the MediaMTX Control API.
type Client struct {
	baseURL string
	auth    *config.ApiAuth
	http    *http.Client
}

// NewClient creates a new MediaMTX API client.
func NewClient(baseURL string, auth *config.ApiAuth) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		auth:    auth,
		http:    &http.Client{},
	}
}

// buildURL normalizes the base URL and endpoint into a full URL.
func (c *Client) buildURL(endpoint string) *url.URL {
	normalizedBase := c.baseURL
	if !strings.HasSuffix(normalizedBase, "/") {
		normalizedBase += "/"
	}
	normalizedEndpoint := endpoint
	if !strings.HasPrefix(normalizedEndpoint, "/") {
		normalizedEndpoint = "/" + normalizedEndpoint
	}

	// Handle v3 path normalization: if base ends with /v3 and endpoint starts with /v3/, strip /v3 from endpoint.
	if strings.HasSuffix(c.baseURL, "/v3") && strings.HasPrefix(normalizedEndpoint, "/v3/") {
		normalizedEndpoint = normalizedEndpoint[3:]
	}

	full := normalizedBase + strings.TrimPrefix(normalizedEndpoint, "/")
	u, _ := url.Parse(full)
	return u
}

// authHeaders returns the Authorization header value.
func (c *Client) authHeaders() http.Header {
	if c.auth == nil {
		return http.Header{}
	}

	if c.auth.Scheme == "bearer" {
		return http.Header{"Authorization": []string{"Bearer " + c.auth.Token}}
	}

	credentials := base64.StdEncoding.EncodeToString(
		[]byte(c.auth.Username + ":" + c.auth.Password),
	)
	return http.Header{"Authorization": []string{"Basic " + credentials}}
}

// request executes an HTTP request with auth headers.
func (c *Client) request(ctx context.Context, u *url.URL, method string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for k, vals := range c.authHeaders() {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// listPaginatedOnce fetches a single page of a paginated API endpoint.
func (c *Client) listPaginatedOnce[T any](ctx context.Context, endpoint string, label string) ([]T, error) {
	var items []T
	page := 0

	for {
		u := c.buildURL(endpoint)
		q := u.Query()
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("itemsPerPage", fmt.Sprintf("%d", constants.PathsPageSize))
		u.RawQuery = q.Encode()

		resp, err := c.request(ctx, u, http.MethodGet)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s failed: %d %s%s", label, resp.StatusCode, resp.Status, authHint(resp.StatusCode))
		}

		var result struct {
			PageCount int `json:"pageCount"`
			ItemCount int `json:"itemCount"`
			Items     []T `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decode %s: %w", label, err)
		}

		items = append(items, result.Items...)
		page++
		if page >= result.PageCount {
			break
		}
	}

	return items, nil
}

// listPaginated fetches all pages, retrying with deprecated endpoint on 404.
func (c *Client) listPaginated[T any](ctx context.Context, endpoint string, label string, optional bool) ([]T, error) {
	candidates := []string{endpoint}
	if dep, ok := constants.DeprecatedAPIPaths[endpoint]; ok {
		candidates = []string{endpoint, dep}
	}

	for i, candidate := range candidates {
		isLast := i == len(candidates)-1
		items, err := c.listPaginatedOnce[T](ctx, candidate, label)
		if err != nil {
			if is404(err) && !isLast {
				continue
			}
			if is404(err) && optional {
				return []T{}, nil
			}
			return nil, err
		}
		return items, nil
	}

	return []T{}, nil
}

// ListPaths fetches all MediaMTX paths.
func (c *Client) ListPaths(ctx context.Context) ([]Path, error) {
	return c.listPaginated[Path](ctx, constants.PathsListEndpoint, "paths/list", false)
}

// ListRtspSessions fetches RTSP sessions (with deprecated fallback).
func (c *Client) ListRtspSessions(ctx context.Context) ([]RtspSession, error) {
	return c.listPaginated[RtspSession](ctx, constants.RtspSessionsList, "rtsp/sessions/list", true)
}

// ListRtspsSessions fetches RTSPS sessions (with deprecated fallback).
func (c *Client) ListRtspsSessions(ctx context.Context) ([]RtspSession, error) {
	return c.listPaginated[RtspSession](ctx, constants.RtspsSessionsList, "rtsps/sessions/list", true)
}

// ListRtmpConns fetches RTMP connections (with deprecated fallback).
func (c *Client) ListRtmpConns(ctx context.Context) ([]RtmpConn, error) {
	return c.listPaginated[RtmpConn](ctx, constants.RtmpConnsList, "rtmp/conns/list", true)
}

// ListRtmpsConns fetches RTMPS connections (with deprecated fallback).
func (c *Client) ListRtmpsConns(ctx context.Context) ([]RtmpConn, error) {
	return c.listPaginated[RtmpConn](ctx, constants.RtmpsConnsList, "rtmps/conns/list", true)
}

// KickPublisher terminates a publisher session by ID.
func (c *Client) KickPublisher(ctx context.Context, source PathSource) error {
	endpoint, ok := constants.KickEndpoints[source.Type]
	if !ok {
		return fmt.Errorf("no kick endpoint for source type %q", source.Type)
	}

	candidates := []string{endpoint}
	if dep, ok := constants.DeprecatedAPIPaths[endpoint]; ok {
		candidates = []string{endpoint, dep}
	}

	var lastErr error
	for i, candidate := range candidates {
		isLast := i == len(candidates)-1
		u := c.buildURL(fmt.Sprintf("%s/%s", candidate, source.ID))

		resp, err := c.request(ctx, u, http.MethodPost)
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var okResp OkResponse
			if err := json.NewDecoder(bytes.NewReader(body)).Decode(&okResp); err != nil || okResp.Status != "ok" {
				return fmt.Errorf("unexpected kick response: %s", string(body))
			}
			return nil
		}

		var errResp ErrorResponse
		_ = json.NewDecoder(bytes.NewReader(body)).Decode(&errResp)
		detail := errResp.Error
		if detail == "" {
			detail = resp.Status
		}
		lastErr = fmt.Errorf("kick failed (%d): %s%s", resp.StatusCode, detail, authHint(resp.StatusCode))

		if resp.StatusCode == http.StatusNotFound && !isLast {
			continue
		}
		return lastErr
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("kick failed: no endpoint tried")
}

func is404(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "404") || strings.Contains(msg, "404 Not Found")
}

func authHint(status int) string {
	if status != http.StatusUnauthorized {
		return ""
	}
	return " (check MTX_API_USER/MTX_API_PASSWORD or MTX_API_TOKEN)"
}
