package requests

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Client struct {
	url    string
	method string
	header http.Header
	params url.Values

	form url.Values
	json interface{}
}

type Result struct {
	Resp *http.Response
	Err  error
}

func Get(url string) *Client {
	c := newClient()
	c.url = url
	c.method = http.MethodGet
	return c
}

func Post(url string) *Client {
	c := newClient()
	c.url = url
	c.method = http.MethodPost
	return c
}

func Put(url string) *Client {
	c := newClient()
	c.url = url
	c.method = http.MethodPut
	return c
}

func Delete(url string) *Client {
	c := newClient()
	c.url = url
	c.method = http.MethodDelete
	return c
}

func (c *Client) Params(params url.Values) *Client {
	for k, v := range params {
		c.params[k] = v
	}
	return c
}

func (c *Client) Header(k, v string) *Client {
	c.header.Set(k, v)
	return c
}

func (c *Client) Headers(header http.Header) *Client {
	for k, v := range header {
		c.header[k] = v
	}
	return c
}

func (c *Client) Form(form url.Values) *Client {
	c.header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.form = form
	return c
}

func (c *Client) Json(json interface{}) *Client {
	c.header.Set("Content-Type", "application/json")
	c.json = json
	return c
}

func (c *Client) Send() *Result {
	r := new(Result)

	if c.params != nil && len(c.params) != 0 {
		c.url += "?" + c.params.Encode()
	}

	contentType := c.header.Get("Content-Type")
	var req *http.Request
	if strings.HasPrefix(contentType, "application/json") {
		b, err := json.Marshal(c.json)
		if err != nil {
			r.Err = err
			return r
		}

		req, r.Err = http.NewRequest(c.method, c.url, bytes.NewReader(b))
		if r.Err != nil {
			return r
		}
	} else if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		form := c.form.Encode()

		req, r.Err = http.NewRequest(c.method, c.url, strings.NewReader(form))
	} else {
		req, r.Err = http.NewRequest(c.method, c.url, nil)
	}

	if r.Err != nil {
		return r
	}

	req.Header = c.header

	r.Resp, r.Err = http.DefaultClient.Do(req)
	if r.Err != nil {
		return r
	}

	return r
}

func (r *Result) Raw() ([]byte, error) {
	if r.Err != nil {
		return nil, r.Err
	}

	b, err := ioutil.ReadAll(r.Resp.Body)
	if err != nil {
		r.Err = err
		return nil, r.Err
	}
	defer r.Resp.Body.Close()

	return b, r.Err
}

func (r *Result) Text() (string, error) {
	b, err := r.Raw()
	if err != nil {
		r.Err = err
		return "", r.Err
	}

	return string(b), nil
}

func (r *Result) Json(v interface{}) error {
	b, err := r.Raw()
	if err != nil {
		r.Err = err
		return r.Err
	}

	return json.Unmarshal(b, v)
}

func (r *Result) Save(name string) error {
	if r.Err != nil {
		return r.Err
	}

	f, err := os.Create(name)
	if err != nil {
		r.Err = err
		return r.Err
	}
	defer f.Close()

	_, err = io.Copy(f, r.Resp.Body)
	if err != nil {
		r.Err = err
		return r.Err
	}

	defer r.Resp.Body.Close()

	return nil
}

func newClient() *Client {
	return &Client{
		header: make(http.Header),
		params: make(url.Values),
		form:   make(url.Values),
	}
}
