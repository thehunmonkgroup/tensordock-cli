package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/caguiclajmg/tensordock-cli/debugutil"
)

const (
	ClientVersion         = "0.9.0"
	defaultHTTPTimeout    = 60 * time.Second
	maxResponseBodyBytes  = 2 << 20
	maxResponseErrorBytes = 4 << 10
)

type Client struct {
	BaseURL    string
	APIToken   string
	Debug      bool
	HTTPClient *http.Client
}

func NewClient(baseURL string, apiToken string, debug bool) (*Client, error) {
	return &Client{
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		APIToken:   apiToken,
		Debug:      debug,
		HTTPClient: newHTTPClient(),
	}, nil
}

func ValidateBaseURL(rawBaseURL string, allowInsecureHTTP bool) (string, error) {
	trimmed := strings.TrimSpace(rawBaseURL)
	if trimmed == "" {
		return "", errors.New("service URL cannot be empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid service URL %q: %w", rawBaseURL, err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", fmt.Errorf("service URL must use http or https: %q", rawBaseURL)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("service URL must include a host: %q", rawBaseURL)
	}
	if parsed.User != nil {
		return "", errors.New("service URL must not include user credentials")
	}
	if parsed.Scheme == "http" && !allowInsecureHTTP {
		return "", errors.New("refusing insecure service URL without allowInsecureHTTP enabled")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawFragment = ""
	parsed.Fragment = ""

	return parsed.String(), nil
}

func newHTTPClient() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{Timeout: defaultHTTPTimeout}
	}

	return &http.Client{
		Timeout:   defaultHTTPTimeout,
		Transport: transport.Clone(),
	}
}

func (client *Client) request(ctx context.Context, method string, endpoint string, query map[string]string, body interface{}, out interface{}) error {
	raw, err := client.do(ctx, method, endpoint, query, body)
	if err != nil {
		return err
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		debugutil.Logf(client.Debug, "api decode failed method=%s endpoint=%s response_bytes=%d err=%v", method, endpoint, len(raw), err)
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func (client *Client) do(ctx context.Context, method string, endpoint string, query map[string]string, body interface{}) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var requestBody io.Reader
	var bodyBytes []byte
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			debugutil.Logf(client.Debug, "request marshal failed method=%s endpoint=%s err=%v", method, endpoint, err)
			return nil, err
		}
		bodyBytes = raw
		requestBody = bytes.NewReader(raw)
	}

	requestURL, err := client.endpointURL(endpoint, query)
	if err != nil {
		debugutil.Logf(client.Debug, "request URL build failed endpoint=%s err=%v", endpoint, err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), requestBody)
	if err != nil {
		debugutil.Logf(client.Debug, "request creation failed method=%s url=%s err=%v", method, debugutil.RedactURL(requestURL.String()), err)
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.APIToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("tensordock-cli/%s", ClientVersion))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	debugutil.Logf(
		client.Debug,
		"api request method=%s endpoint=%s url=%s query=%s body_bytes=%d body_redacted=%t",
		method,
		endpoint,
		debugutil.RedactURL(req.URL.String()),
		debugutil.FormatStringMap(query),
		len(bodyBytes),
		len(bodyBytes) > 0,
	)

	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = newHTTPClient()
	}

	startedAt := time.Now()
	res, err := httpClient.Do(req)
	if err != nil {
		debugutil.Logf(client.Debug, "api transport error method=%s endpoint=%s url=%s duration=%s err=%v", method, endpoint, debugutil.RedactURL(req.URL.String()), time.Since(startedAt), err)
		return nil, err
	}
	defer res.Body.Close()

	raw, err := readLimitedBody(res.Body, maxResponseBodyBytes)
	if err != nil {
		debugutil.Logf(client.Debug, "api response read failed method=%s endpoint=%s status=%s duration=%s err=%v", method, endpoint, res.Status, time.Since(startedAt), err)
		return nil, err
	}

	debugutil.Logf(client.Debug, "api response method=%s endpoint=%s status=%s duration=%s response_bytes=%d body_redacted=%t", method, endpoint, res.Status, time.Since(startedAt), len(raw), len(raw) > 0)

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		debugutil.Logf(client.Debug, "api non-success status method=%s endpoint=%s status=%s", method, endpoint, res.Status)
		if len(raw) == 0 {
			return nil, fmt.Errorf("api request failed with status %s", res.Status)
		}
		return nil, fmt.Errorf("api request failed with status %s: %s", res.Status, truncateForError(apiErrorMessage(raw), maxResponseErrorBytes))
	}

	if err := apiErrorFromBody(raw); err != nil {
		debugutil.Logf(client.Debug, "api application error method=%s endpoint=%s status=%s err=%v", method, endpoint, res.Status, err)
		return nil, err
	}

	return raw, nil
}

func (client *Client) endpointURL(endpoint string, query map[string]string) (*url.URL, error) {
	baseURL, err := url.Parse(client.BaseURL)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(baseURL.Path, endpoint)

	values := baseURL.Query()
	for key, value := range query {
		if value == "" {
			continue
		}
		values.Set(key, value)
	}
	baseURL.RawQuery = values.Encode()

	return baseURL, nil
}

func readLimitedBody(body io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(body, maxBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes", maxBytes)
	}

	return raw, nil
}

func apiErrorFromBody(raw []byte) error {
	if len(raw) == 0 {
		return nil
	}

	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Status  int    `json:"status"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	message := strings.TrimSpace(payload.Error)
	if message == "" {
		message = strings.TrimSpace(payload.Message)
	}
	if message == "" {
		return nil
	}

	if payload.Status >= 400 || payload.Error != "" {
		if payload.Status > 0 {
			return fmt.Errorf("api request failed with status %d: %s", payload.Status, truncateForError(message, maxResponseErrorBytes))
		}
		return errors.New(truncateForError(message, maxResponseErrorBytes))
	}

	return nil
}

func apiErrorMessage(raw []byte) string {
	if err := apiErrorFromBody(raw); err != nil {
		return err.Error()
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "upstream returned an empty error response"
	}

	return trimmed
}

func truncateForError(value string, maxBytes int) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= maxBytes {
		return trimmed
	}
	if maxBytes < 4 {
		return trimmed[:maxBytes]
	}

	return trimmed[:maxBytes-3] + "..."
}

func isTimeoutError(err error) bool {
	type timeout interface {
		Timeout() bool
	}

	var netErr timeout
	return errors.As(err, &netErr) && netErr.Timeout()
}

func (client *Client) ListSecrets(ctx context.Context) ([]SecretSummary, error) {
	var response struct {
		Data struct {
			Secrets []SecretSummary `json:"secrets"`
		} `json:"data"`
	}

	if err := client.request(ctx, http.MethodGet, "secrets", nil, nil, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return response.Data.Secrets, nil
}

func (client *Client) CreateSecret(ctx context.Context, request SecretCreateRequest) (*Secret, error) {
	var response struct {
		Data Secret `json:"data"`
	}

	if err := client.request(ctx, http.MethodPost, "secrets", nil, request, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return &response.Data, nil
}

func (client *Client) GetSecret(ctx context.Context, id string) (*Secret, error) {
	var response struct {
		Data Secret `json:"data"`
	}

	if err := client.request(ctx, http.MethodGet, fmt.Sprintf("secrets/%s", id), nil, nil, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return &response.Data, nil
}

func (client *Client) DeleteSecret(ctx context.Context, id string) (*MessageResponse, error) {
	var response MessageResponse

	if err := client.request(ctx, http.MethodDelete, fmt.Sprintf("secrets/%s", id), nil, nil, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return &response, nil
}

func (client *Client) ListLocations(ctx context.Context) ([]Location, error) {
	var response struct {
		Data struct {
			Locations []Location `json:"locations"`
		} `json:"data"`
	}

	if err := client.request(ctx, http.MethodGet, "locations", nil, nil, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return response.Data.Locations, nil
}

func (client *Client) ListHostnodes(ctx context.Context, filters map[string]string) ([]Hostnode, error) {
	var response struct {
		Data struct {
			Hostnodes []Hostnode `json:"hostnodes"`
		} `json:"data"`
	}

	if err := client.request(ctx, http.MethodGet, "hostnodes", filters, nil, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return response.Data.Hostnodes, nil
}

func (client *Client) GetHostnode(ctx context.Context, id string) (*Hostnode, error) {
	var response struct {
		Data Hostnode `json:"data"`
	}

	if err := client.request(ctx, http.MethodGet, fmt.Sprintf("hostnodes/%s", id), nil, nil, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return &response.Data, nil
}

func (client *Client) ListInstances(ctx context.Context) ([]InstanceListItem, error) {
	var response struct {
		Data json.RawMessage `json:"data"`
	}

	if err := client.request(ctx, http.MethodGet, "instances", nil, nil, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	var direct []InstanceListItem
	if err := json.Unmarshal(response.Data, &direct); err == nil {
		debugutil.Logf(client.Debug, "list instances used response path=data count=%d", len(direct))
		return direct, nil
	}

	var wrapped struct {
		Instances  []InstanceListItem `json:"instances"`
		Attributes struct {
			Instances []InstanceListItem `json:"instances"`
		} `json:"attributes"`
	}
	if err := json.Unmarshal(response.Data, &wrapped); err != nil {
		return nil, fmt.Errorf("decode instances data: %w", err)
	}

	if len(wrapped.Instances) > 0 {
		debugutil.Logf(client.Debug, "list instances used response path=data.instances count=%d", len(wrapped.Instances))
		return wrapped.Instances, nil
	}

	debugutil.Logf(client.Debug, "list instances used response path=data.attributes.instances count=%d", len(wrapped.Attributes.Instances))
	return wrapped.Attributes.Instances, nil
}

func (client *Client) GetInstance(ctx context.Context, id string) (*Instance, error) {
	raw, err := client.do(ctx, http.MethodGet, fmt.Sprintf("instances/%s", id), nil, nil)
	if err != nil {
		return nil, normalizeTransportError(err)
	}

	var wrapped struct {
		Data Instance `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Data.ID != "" {
		debugutil.Logf(client.Debug, "get instance used wrapped response id=%s", id)
		return &wrapped.Data, nil
	}

	var direct Instance
	if err := json.Unmarshal(raw, &direct); err != nil {
		return nil, fmt.Errorf("decode instance: %w", err)
	}

	debugutil.Logf(client.Debug, "get instance used direct response id=%s", id)
	return &direct, nil
}

func (client *Client) CreateInstance(ctx context.Context, request InstanceCreateRequest) (*InstanceCreateResponse, error) {
	var response struct {
		Data InstanceCreateResponse `json:"data"`
	}

	if err := client.request(ctx, http.MethodPost, "instances", nil, request, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return &response.Data, nil
}

func (client *Client) StartInstance(ctx context.Context, id string) (*ActionStatusResponse, error) {
	var response ActionStatusResponse

	if err := client.request(ctx, http.MethodPost, fmt.Sprintf("instances/%s/start", id), nil, nil, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return &response, nil
}

func (client *Client) StopInstance(ctx context.Context, id string) (*ActionStatusResponse, error) {
	var response ActionStatusResponse

	if err := client.request(ctx, http.MethodPost, fmt.Sprintf("instances/%s/stop", id), nil, nil, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return &response, nil
}

func (client *Client) DeleteInstance(ctx context.Context, id string) (*MessageResponse, error) {
	var response MessageResponse

	if err := client.request(ctx, http.MethodDelete, fmt.Sprintf("instances/%s", id), nil, nil, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return &response, nil
}

func (client *Client) ModifyInstance(ctx context.Context, id string, request InstanceModifyRequest) (*MessageResponse, error) {
	var response MessageResponse

	if err := client.request(ctx, http.MethodPut, fmt.Sprintf("instances/%s/modify", id), nil, request, &response); err != nil {
		return nil, normalizeTransportError(err)
	}

	return &response, nil
}

func normalizeTransportError(err error) error {
	if err == nil {
		return nil
	}
	if isTimeoutError(err) {
		return fmt.Errorf("request timed out: %w", err)
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("request timed out: %w", err)
	}

	return err
}
