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
	this := &Proxmox{}

	return this
}

func (this *Proxmox) SetAuth(login, token string) {
	this.token = fmt.Sprintf("%s!%s", login, token)
}

func (this *Proxmox) AddNode(node string) {
	this.nodes = append(this.nodes, node)
}

func (this *Proxmox) getNodeURL() string {
	return this.nodes[0]
}

func joinPath(a, b string) string {
	u, err := url.Parse(a)
	if err != nil {
		log.Fatal(err)
	}

	u.Path = path.Join(u.Path, "api2/json", b)

	return u.String()
}

func (this *Proxmox) makeHTTPRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, joinPath(this.getNodeURL(), url), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("PVEAPIToken=%s", this.token))

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
