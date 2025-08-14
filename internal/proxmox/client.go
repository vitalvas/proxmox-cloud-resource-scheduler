package proxmox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/vitalvas/proxmox-cloud-resource-scheduler/internal/logging"
)

type Client struct {
	config     *Config
	httpClient *http.Client
	authTicket string
	csrfToken  string
}

type APIResponse struct {
	Data interface{} `json:"data"`
}

type APIError struct {
	Status  int    `json:"status"`
	Message string `json:"error"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Status, e.Message)
}

func NewClient(config *Config) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.TLS.InsecureSkipVerify,
		},
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   config.Timeout,
		},
	}
}

func (c *Client) getRandomEndpoint() string {
	if len(c.config.Endpoints) == 0 {
		return ""
	}
	return c.config.Endpoints[rand.Intn(len(c.config.Endpoints))]
}

func (c *Client) buildURL(endpoint string) string {
	baseURL := c.getRandomEndpoint()
	if baseURL == "" {
		return ""
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		logging.Errorf("Failed to parse base URL %s: %v", baseURL, err)
		return ""
	}

	u.Path = path.Join(u.Path, "api2/json", endpoint)
	return u.String()
}

func (c *Client) authenticate() error {
	if c.config.Auth.Method == "token" {
		return nil
	}

	data := url.Values{}
	data.Set("username", c.config.Auth.Username+"@"+c.config.Auth.Realm)
	data.Set("password", c.config.Auth.Password)

	resp, err := c.makeRequest("POST", "access/ticket", strings.NewReader(data.Encode()), false)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	defer resp.Body.Close()

	var authResp struct {
		Data struct {
			Ticket    string `json:"ticket"`
			CSRFToken string `json:"CSRFPreventionToken"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	c.authTicket = authResp.Data.Ticket
	c.csrfToken = authResp.Data.CSRFToken

	logging.Debug("Successfully authenticated with Proxmox")
	return nil
}

func (c *Client) makeRequest(method, endpoint string, body io.Reader, auth bool) (*http.Response, error) {
	url := c.buildURL(endpoint)
	if url == "" {
		return nil, fmt.Errorf("failed to build URL for endpoint: %s", endpoint)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if auth {
		if err := c.setAuthHeaders(req); err != nil {
			return nil, fmt.Errorf("failed to set auth headers: %w", err)
		}
	}

	if method == http.MethodPost || method == http.MethodPut {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	logging.Debugf("Making %s request to %s", method, url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()

		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	return resp, nil
}

func (c *Client) setAuthHeaders(req *http.Request) error {
	switch c.config.Auth.Method {
	case "token":
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", c.config.Auth.APIToken))
	case "password":
		if c.authTicket == "" {
			if err := c.authenticate(); err != nil {
				return err
			}
		}
		req.Header.Set("Cookie", fmt.Sprintf("PVEAuthCookie=%s", c.authTicket))
		if req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodDelete {
			req.Header.Set("CSRFPreventionToken", c.csrfToken)
		}
	default:
		return fmt.Errorf("unsupported auth method: %s", c.config.Auth.Method)
	}
	return nil
}

func (c *Client) Get(endpoint string, result interface{}) error {
	resp, err := c.makeRequest("GET", endpoint, nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.parseResponse(resp, result)
}

func (c *Client) Post(endpoint string, data url.Values, result interface{}) error {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}

	resp, err := c.makeRequest("POST", endpoint, body, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.parseResponse(resp, result)
}

func (c *Client) Put(endpoint string, data url.Values, result interface{}) error {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}

	resp, err := c.makeRequest("PUT", endpoint, body, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.parseResponse(resp, result)
}

func (c *Client) Delete(endpoint string) error {
	resp, err := c.makeRequest("DELETE", endpoint, nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) DeleteWithResponse(endpoint string, result interface{}) error {
	resp, err := c.makeRequest("DELETE", endpoint, nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.parseResponse(resp, result)
}

func (c *Client) parseResponse(resp *http.Response, result interface{}) error {
	if result == nil {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp APIResponse
	apiResp.Data = result

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

func (c *Client) createBackup(resourceType, node string, vmid int, options BackupOptions) (string, error) {
	endpoint := fmt.Sprintf("nodes/%s/%s/%d/backup", node, resourceType, vmid)
	data := url.Values{}
	data.Set("storage", options.Storage)

	if options.Mode != "" {
		data.Set("mode", options.Mode)
	}
	if options.Compress != "" {
		data.Set("compress", options.Compress)
	}
	if options.MailTo != "" {
		data.Set("mailto", options.MailTo)
	}
	if options.Notes != "" {
		data.Set("notes", options.Notes)
	}
	if options.Protected {
		data.Set("protected", "1")
	}

	var taskID string
	if err := c.Post(endpoint, data, &taskID); err != nil {
		return "", fmt.Errorf("failed to create backup for %s %d on node %s: %w", resourceType, vmid, node, err)
	}

	logging.Infof("Created backup for %s %d on node %s, task: %s", resourceType, vmid, node, taskID)
	return taskID, nil
}
