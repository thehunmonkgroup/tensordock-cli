package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/caguiclajmg/tensordock-cli/debugutil"
)

var CLIENT_VERSION = "0.8.0"

type Client struct {
	BaseURL    string
	APIToken   string
	Debug      bool
	HTTPClient *http.Client
}

func NewClient(baseURL string, apiToken string, debug bool) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIToken:   apiToken,
		Debug:      debug,
		HTTPClient: http.DefaultClient,
	}
}

func (client *Client) request(method string, endpoint string, query map[string]string, body interface{}, out interface{}) error {
	var requestBody io.Reader
	var bodyBytes []byte
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			debugutil.Logf(client.Debug, "request marshal failed method=%s endpoint=%s err=%v", method, endpoint, err)
			return err
		}
		bodyBytes = raw
		requestBody = bytes.NewReader(raw)
	}

	baseURL, err := url.Parse(client.BaseURL)
	if err != nil {
		debugutil.Logf(client.Debug, "invalid base URL %q: %v", client.BaseURL, err)
		return err
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

	req, err := http.NewRequest(method, baseURL.String(), requestBody)
	if err != nil {
		debugutil.Logf(client.Debug, "request creation failed method=%s url=%s err=%v", method, debugutil.RedactURL(baseURL.String()), err)
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.APIToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("tensordock-cli/%s", CLIENT_VERSION))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	debugutil.Logf(
		client.Debug,
		"api request method=%s endpoint=%s url=%s query=%s body_bytes=%d body=%s",
		method,
		endpoint,
		debugutil.RedactURL(req.URL.String()),
		debugutil.FormatStringMap(query),
		len(bodyBytes),
		debugutil.RedactJSONBytes(bodyBytes),
	)

	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	startedAt := time.Now()
	res, err := httpClient.Do(req)
	if err != nil {
		debugutil.Logf(client.Debug, "api transport error method=%s endpoint=%s url=%s duration=%s err=%v", method, endpoint, debugutil.RedactURL(req.URL.String()), time.Since(startedAt), err)
		return err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		debugutil.Logf(client.Debug, "api response read failed method=%s endpoint=%s status=%s duration=%s err=%v", method, endpoint, res.Status, time.Since(startedAt), err)
		return err
	}

	debugutil.Logf(client.Debug, "api response method=%s endpoint=%s status=%s duration=%s response_bytes=%d body=%s", method, endpoint, res.Status, time.Since(startedAt), len(raw), debugutil.RedactJSONBytes(raw))

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		debugutil.Logf(client.Debug, "api non-success status method=%s endpoint=%s status=%s", method, endpoint, res.Status)
		if len(raw) == 0 {
			return fmt.Errorf("api request failed with status %s", res.Status)
		}
		return fmt.Errorf("api request failed with status %s: %s", res.Status, strings.TrimSpace(string(raw)))
	}

	if err := apiErrorFromBody(raw); err != nil {
		debugutil.Logf(client.Debug, "api application error method=%s endpoint=%s status=%s err=%v", method, endpoint, res.Status, err)
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
			return fmt.Errorf("api request failed with status %d: %s", payload.Status, message)
		}
		return errors.New(message)
	}

	return nil
}

func (client *Client) ListSecrets() ([]SecretSummary, error) {
	var response struct {
		Data struct {
			Secrets []SecretSummary `json:"secrets"`
		} `json:"data"`
	}

	if err := client.request(http.MethodGet, "secrets", nil, nil, &response); err != nil {
		return nil, err
	}

	return response.Data.Secrets, nil
}

func (client *Client) CreateSecret(request SecretCreateRequest) (*Secret, error) {
	var response struct {
		Data Secret `json:"data"`
	}

	if err := client.request(http.MethodPost, "secrets", nil, request, &response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (client *Client) GetSecret(id string) (*Secret, error) {
	var response struct {
		Data Secret `json:"data"`
	}

	if err := client.request(http.MethodGet, fmt.Sprintf("secrets/%s", id), nil, nil, &response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (client *Client) DeleteSecret(id string) (*MessageResponse, error) {
	var response MessageResponse

	if err := client.request(http.MethodDelete, fmt.Sprintf("secrets/%s", id), nil, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (client *Client) ListLocations() ([]Location, error) {
	var response struct {
		Data struct {
			Locations []Location `json:"locations"`
		} `json:"data"`
	}

	if err := client.request(http.MethodGet, "locations", nil, nil, &response); err != nil {
		return nil, err
	}

	return response.Data.Locations, nil
}

func (client *Client) ListHostnodes(filters map[string]string) ([]Hostnode, error) {
	var response struct {
		Data struct {
			Hostnodes []Hostnode `json:"hostnodes"`
		} `json:"data"`
	}

	if err := client.request(http.MethodGet, "hostnodes", filters, nil, &response); err != nil {
		return nil, err
	}

	return response.Data.Hostnodes, nil
}

func (client *Client) GetHostnode(id string) (*Hostnode, error) {
	var response struct {
		Data Hostnode `json:"data"`
	}

	if err := client.request(http.MethodGet, fmt.Sprintf("hostnodes/%s", id), nil, nil, &response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (client *Client) ListInstances() ([]InstanceListItem, error) {
	var response struct {
		Data json.RawMessage `json:"data"`
	}

	if err := client.request(http.MethodGet, "instances", nil, nil, &response); err != nil {
		return nil, err
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

func (client *Client) GetInstance(id string) (*Instance, error) {
	var wrapped struct {
		Data Instance `json:"data"`
	}
	if err := client.request(http.MethodGet, fmt.Sprintf("instances/%s", id), nil, nil, &wrapped); err == nil && wrapped.Data.ID != "" {
		debugutil.Logf(client.Debug, "get instance used wrapped response id=%s", id)
		return &wrapped.Data, nil
	}
	debugutil.Logf(client.Debug, "get instance falling back to direct response id=%s", id)

	var direct Instance
	if err := client.request(http.MethodGet, fmt.Sprintf("instances/%s", id), nil, nil, &direct); err != nil {
		return nil, err
	}

	return &direct, nil
}

func (client *Client) CreateInstance(request InstanceCreateRequest) (*InstanceCreateResponse, error) {
	var response struct {
		Data InstanceCreateResponse `json:"data"`
	}

	if err := client.request(http.MethodPost, "instances", nil, request, &response); err != nil {
		return nil, err
	}

	return &response.Data, nil
}

func (client *Client) StartInstance(id string) (*ActionStatusResponse, error) {
	var response ActionStatusResponse

	if err := client.request(http.MethodPost, fmt.Sprintf("instances/%s/start", id), nil, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (client *Client) StopInstance(id string) (*ActionStatusResponse, error) {
	var response ActionStatusResponse

	if err := client.request(http.MethodPost, fmt.Sprintf("instances/%s/stop", id), nil, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (client *Client) DeleteInstance(id string) (*MessageResponse, error) {
	var response MessageResponse

	if err := client.request(http.MethodDelete, fmt.Sprintf("instances/%s", id), nil, nil, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (client *Client) ModifyInstance(id string, request InstanceModifyRequest) (*MessageResponse, error) {
	var response MessageResponse

	if err := client.request(http.MethodPut, fmt.Sprintf("instances/%s/modify", id), nil, request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
