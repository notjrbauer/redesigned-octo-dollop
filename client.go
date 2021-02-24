package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
)

// Client returns a docker communicating client
type Client struct {
	tags map[string][]string
	*http.Client
}

func NewClient(tags map[string][]string, socketPath string) Client {
	httpc := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
	return Client{tags, httpc}
}

type ports struct {
	Type        string  `json:"Type,omitempty"`
	IP          string  `json:"IP,omitempty"`
	PrivatePort float64 `json:"PrivatePort,omitempty"`
	PublicPort  float64 `json:"PublicPort,omitempty"`
}

// Returns the service's upstream SRV entries.
func (c *Client) GetAddrs(service string) ([]net.SRV, error) {
	out := []net.SRV{}
	query := url.Values{}

	tags := c.tags[service]
	if tags == nil {
		return out, errors.New("not found")
	}
	m := map[string][]string{"label": tags}
	filterJSON, err := ToJSON(m)
	if err != nil {
		return out, err
	}
	query.Set("filters", filterJSON)

	resp, err := c.Get("http://::/containers/json?" + query.Encode())
	if err != nil {
		return out, err
	}
	bbytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return out, err
	}
	var o []struct{ Ports []ports }
	if err := json.Unmarshal(bbytes, &o); err != nil {
		return out, err
	}
	for _, s := range o {
		for _, r := range s.Ports {
			if r.PublicPort > 0 {
				out = append(out, net.SRV{
					Target:   r.IP,
					Port:     uint16(r.PublicPort),
					Weight:   uint16(0),
					Priority: uint16(0),
				})
			}
		}
	}
	return out, nil
}

func ToJSON(i interface{}) (string, error) {
	if i == nil {
		return "", nil
	}
	buf, err := json.Marshal(i)
	return string(buf), err
}
