package requests

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
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

	form      url.Values
	json      interface{}
	multipart FileForm
}

type FileForm struct {
	Value url.Values
	File  map[string]string
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

func (c *Client) Multipart(multipart FileForm) *Client {
	c.multipart = multipart
	return c
}

func (c *Client) Send() *Result {
	var result *Result

	if c.params != nil && len(c.params) != 0 {
		c.url += "?" + c.params.Encode()
	}

	contentType := c.header.Get("Content-Type")
	if c.multipart.Value != nil || c.multipart.File != nil {
		result = c.createMultipartForm()
	} else if strings.HasPrefix(contentType, "application/json") {
		result = c.createJson()
	} else if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		result = c.createForm()
	} else {
		var result = new(Result)

		req, err := http.NewRequest(c.method, c.url, nil)
		if err != nil {
			result.Err = err
			return result
		}

		req.Header = c.header
		result.Resp, result.Err = http.DefaultClient.Do(req)
		return result
	}

	return result
}

// form-data
func (c *Client) createMultipartForm() *Result {
	var result = new(Result)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for name, filename := range c.multipart.File {
		file, err := os.Open(filename)
		if err != nil {
			result.Err = err
			return result
		}

		part, err := writer.CreateFormFile(name, filename)
		if err != nil {
			result.Err = err
			return result
		}

		// todo 这里的io.Copy实现，会把file文件都读取到内存里面，然后当做一个buffer传给NewRequest。对于大文件来说会占用很多内存
		_, err = io.Copy(part, file)
		if err != nil {
			result.Err = err
			return result
		}

		err = file.Close()
		if err != nil {
			result.Err = err
			return result
		}
	}

	for name, values := range c.multipart.Value {
		for _, value := range values {
			_ = writer.WriteField(name, value)
		}
	}

	err := writer.Close()
	if err != nil {
		result.Err = err
		return result
	}

	req, err := http.NewRequest(c.method, c.url, body)
	req.Header = c.header
	req.Header.Set("Content-Type", writer.FormDataContentType())
	result.Resp, result.Err = http.DefaultClient.Do(req)
	return result
}

// application/json
func (c *Client) createJson() *Result {
	var result = new(Result)

	b, err := json.Marshal(c.json)
	if err != nil {
		result.Err = err
		return result
	}

	req, err := http.NewRequest(c.method, c.url, bytes.NewReader(b))
	if err != nil {
		result.Err = err
		return result
	}

	req.Header = c.header
	result.Resp, result.Err = http.DefaultClient.Do(req)
	return result
}

// application/x-www-form-urlencoded
func (c *Client) createForm() *Result {
	var result = new(Result)

	form := c.form.Encode()

	req, err := http.NewRequest(c.method, c.url, strings.NewReader(form))
	if err != nil {
		result.Err = err
		return result
	}

	req.Header = c.header
	result.Resp, result.Err = http.DefaultClient.Do(req)
	return result
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
