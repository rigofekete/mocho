package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type PageRef struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

type Page struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	return &Client{BaseURL: baseURL, HTTP: http.DefaultClient}
}

type listResponse struct {
	Pages []PageRef `json:"pages"`
}

func (c *Client) ListPages(ctx context.Context) ([]PageRef, error) {
	resp, body, err := c.get(ctx, "/api/pages")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list pages: %s", resp.Status)
	}
	var out listResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.Pages == nil {
		out.Pages = []PageRef{}
	}
	return out.Pages, nil
}

func (c *Client) ReadPage(ctx context.Context, name string) (Page, error) {
	resp, body, err := c.get(ctx, "/api/pages/"+url.PathEscape(name))
	if err != nil {
		return Page{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e struct{ Error string `json:"error"` }
		_ = json.Unmarshal(body, &e)
		if e.Error != "" {
			return Page{}, fmt.Errorf("read page: %s", e.Error)
		}
		return Page{}, fmt.Errorf("read page: %s", resp.Status)
	}
	var p Page
	if err := json.Unmarshal(body, &p); err != nil {
		return Page{}, err
	}
	return p, nil
}

func (c *Client) get(ctx context.Context, path string) (*http.Response, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, err
	}
	body, _ := io.ReadAll(resp.Body)
	return resp, body, nil
}