package proxmox

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"path"
)

type Proxmox struct {
	GetToken    func() (string, error)
	GetNodesURL func() ([]string, error)
}

func New() *Proxmox {
	return &Proxmox{}
}

func (p *Proxmox) getNodeURL() (string, error) {
	list, err := p.GetNodesURL()
	if err != nil {
		return "", err
	}

	n := rand.Intn(len(list))

	return list[n], nil
}

func joinPath(a, b string) string {
	u, err := url.Parse(a)
	if err != nil {
		log.Fatal(err)
	}

	u.Path = path.Join(u.Path, "api2/json", b)

	return u.String()
}

func (p *Proxmox) makeHTTPRequest(method, url string, body io.Reader) (*http.Response, error) {
	nodeURL, err := p.getNodeURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get node URL: %w", err)
	}

	req, err := http.NewRequest(method, joinPath(nodeURL, url), body)
	if err != nil {
		return nil, err
	}

	token, err := p.GetToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("PVEAPIToken=%s", token))

	if method == http.MethodPost || method == http.MethodPut {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return httpClient.Do(req)
}
