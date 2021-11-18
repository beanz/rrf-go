/*
Copyright (c) 2021 Mark Hindess

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package netrrf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/beanz/rrf-go/pkg/types"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	host       string
	password   string
	authDone   bool
	timeout    time.Duration
	httpClient HttpClient
}

func NewClient(host, password string) *Client {
	return &Client{host, password, false, 30 * time.Second, http.DefaultClient}
}

func (c *Client) WithTimeout(t time.Duration) *Client {
	c.timeout = t
	return c
}

func (c *Client) WithHTTPClient(client HttpClient) *Client {
	c.httpClient = client
	return c
}

func (c *Client) Request(ctx context.Context, uri string, res interface{}) error {
	ctx, cancel := context.WithCancel(ctx)
	timer := time.AfterFunc(c.timeout, func() {
		cancel()
	})

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("http://%s/%s", c.host, uri), nil)
	if err != nil {
		return fmt.Errorf(
			"rrf request creation failed for host %s: %w",
			c.host, err)
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf(
			"rrf request failed for host %s: %w",
			c.host, err)
	}
	defer resp.Body.Close()
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf(
			"rrf response read failed for host %s: %w",
			c.host, err)
	}
	err = json.Unmarshal(buf, res)
	if err != nil {
		return fmt.Errorf(
			"rrf response unmarshal failed for host %s: %w",
			c.host, err)
	}
	timer.Stop()
	return nil
}

type AuthenticationError types.AuthResponse

func (err AuthenticationError) Error() string {
	return fmt.Sprintf("authentication failed with error code=%d",
		err.ErrorCode)
}

func (c *Client) Authenticate(ctx context.Context) error {
	var resp types.AuthResponse
	err := c.Request(ctx, "rr_connect?password="+c.password, &resp)
	if err != nil {
		return fmt.Errorf("rrf auth failed %w", err)
	}
	if resp.ErrorCode != 0 {
		return AuthenticationError(resp)
	}
	c.authDone = true
	return nil
}

func (c *Client) Config(ctx context.Context) (*types.ConfigResponse, error) {
	if !c.authDone {
		err := c.Authenticate(ctx)
		if err != nil {
			return nil, err
		}
	}
	var resp types.ConfigResponse
	err := c.Request(ctx, "rr_config", &resp)
	if err != nil {
		return nil, fmt.Errorf("rrf config failed %w", err)
	}
	return &resp, nil
}

func (c *Client) Status(ctx context.Context, t int) (*types.StatusResponse, error) {
	if !c.authDone {
		err := c.Authenticate(ctx)
		if err != nil {
			return nil, err
		}
	}
	var res types.StatusResponse
	err := c.Request(ctx, fmt.Sprintf("rr_status?type=%d", t), &res)
	if err != nil {
		return nil, fmt.Errorf("rrf status %d failed %w", t, err)
	}
	return &res, nil
}
