package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
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
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestBody = bytes.NewReader(raw)
	}

	baseURL, err := url.Parse(client.BaseURL)
	if err != nil {
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
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.APIToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("tensordock-cli/%s", CLIENT_VERSION))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if client.Debug {
		reqDump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Println(string(reqDump))
	}

	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if client.Debug {
		resDump, err := httputil.DumpResponse(res, true)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Println(string(resDump))
	}

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if len(raw) == 0 {
			return fmt.Errorf("api request failed with status %s", res.Status)
		}
		return fmt.Errorf("api request failed with status %s: %s", res.Status, strings.TrimSpace(string(raw)))
	}

	if out == nil || len(raw) == 0 {
		return nil
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
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
		Data struct {
			Instances  []InstanceListItem `json:"instances"`
			Attributes struct {
				Instances []InstanceListItem `json:"instances"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := client.request(http.MethodGet, "instances", nil, nil, &response); err != nil {
		return nil, err
	}

	if len(response.Data.Instances) > 0 {
		return response.Data.Instances, nil
	}

	return response.Data.Attributes.Instances, nil
}

func (client *Client) GetInstance(id string) (*Instance, error) {
	var wrapped struct {
		Data Instance `json:"data"`
	}
	if err := client.request(http.MethodGet, fmt.Sprintf("instances/%s", id), nil, nil, &wrapped); err == nil && wrapped.Data.ID != "" {
		return &wrapped.Data, nil
	}

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
