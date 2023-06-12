package proxmox

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
)

type Proxmox struct {
	nodes []string
	token string
}

func New() *Proxmox {
	return &Proxmox{}
}

func (p *Proxmox) SetAuth(login, token string) {
	p.token = fmt.Sprintf("%s!%s", login, token)
}

func (p *Proxmox) AddNode(node string) {
	p.nodes = append(p.nodes, node)
}

func (p *Proxmox) getNodeURL() string {
	return p.nodes[0]
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
	req, err := http.NewRequest(method, joinPath(p.getNodeURL(), url), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("PVEAPIToken=%s", p.token))

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
