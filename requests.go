package requests

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Client: 封装了http的参数等信息
type Client struct {
	// 自定义Client
	client *http.Client

	url    string
	method string
	header http.Header
	params url.Values

	form      url.Values
	json      interface{}
	multipart FileForm
}

// FileForm: form参数和文件参数
type FileForm struct {
	Value url.Values
	File  map[string]string
}

// Result: http响应结果
type Result struct {
	Resp *http.Response
	Err  error
}

// Get: http `GET` 请求
func Get(url string) *Client {
	c := newClient()
	c.url = url
	c.method = http.MethodGet
	return c
}

// Post: http `POST` 请求
func Post(url string) *Client {
	c := newClient()
	c.url = url
	c.method = http.MethodPost
	return c
}

// Put: http `PUT` 请求
func Put(url string) *Client {
	c := newClient()
	c.url = url
	c.method = http.MethodPut
	return c
}

// Delete: http `DELETE` 请求
func Delete(url string) *Client {
	c := newClient()
	c.url = url
	c.method = http.MethodDelete
	return c
}

// Request: 用于自定义请求方式，比如`HEAD`、`PATCH`、`OPTIONS`、`TRACE`
// client参数用于替换DefaultClient，如果为nil则会使用默认的
func Request(url, method string, client *http.Client) *Client {
	c := newClient()
	c.client = client
	c.url = url
	c.method = method
	return c
}

// Params: http请求中url参数
func (c *Client) Params(params url.Values) *Client {
	for k, v := range params {
		c.params[k] = v
	}
	return c
}

// Header: http请求头
func (c *Client) Header(k, v string) *Client {
	c.header.Set(k, v)
	return c
}

// Headers: http请求头
func (c *Client) Headers(header http.Header) *Client {
	for k, v := range header {
		c.header[k] = v
	}
	return c
}

// Form: 表单提交参数
func (c *Client) Form(form url.Values) *Client {
	c.header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.form = form
	return c
}

// Json: json提交参数
func (c *Client) Json(json interface{}) *Client {
	c.header.Set("Content-Type", "application/json")
	c.json = json
	return c
}

// Multipart: form-data提交参数
func (c *Client) Multipart(multipart FileForm) *Client {
	c.multipart = multipart
	return c
}

// Send: 发送http请求
func (c *Client) Send() *Result {
	var result *Result

	if c.params != nil && len(c.params) != 0 {
		// 如果url中已经有query string参数，则只需要&拼接剩下的即可
		encoded := c.params.Encode()
		if strings.Index(c.url, "?") == -1 {
			c.url += "?" + encoded
		} else {
			c.url += "&" + encoded
		}
	}

	contentType := c.header.Get("Content-Type")
	if c.multipart.Value != nil || c.multipart.File != nil {
		result = c.createMultipartForm()
	} else if strings.HasPrefix(contentType, "application/json") {
		result = c.createJson()
	} else if strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		result = c.createForm()
	} else {
		result = c.createEmptyBody()
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
	c.doSend(req, result)
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
	c.doSend(req, result)
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
	c.doSend(req, result)
	return result
}

// none http body
func (c *Client) createEmptyBody() *Result {
	var result = new(Result)

	req, err := http.NewRequest(c.method, c.url, nil)
	if err != nil {
		result.Err = err
		return result
	}

	req.Header = c.header
	c.doSend(req, result)
	return result
}

func (c *Client) doSend(req *http.Request, result *Result) {
	if c.client != nil {
		result.Resp, result.Err = c.client.Do(req)
		return
	}

	result.Resp, result.Err = http.DefaultClient.Do(req)
}

// StatusOk: 判断http响应码是否为200
func (r *Result) StatusOk() *Result {
	if r.Err != nil {
		return r
	}
	if r.Resp.StatusCode != http.StatusOK {
		r.Err = errors.New("status code is not 200")
		return r
	}

	return r
}

// Status2xx: 判断http响应码是否为2xx
func (r *Result) Status2xx() *Result {
	if r.Err != nil {
		return r
	}
	if r.Resp.StatusCode < http.StatusOK || r.Resp.StatusCode >= http.StatusMultipleChoices {
		r.Err = errors.New("status code is not match [200, 300)")
		return r
	}

	return r
}

// Raw: 获取http响应内容，返回字节数组
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

// Text: 获取http响应内容，返回字符串
func (r *Result) Text() (string, error) {
	b, err := r.Raw()
	if err != nil {
		r.Err = err
		return "", r.Err
	}

	return string(b), nil
}

// Json: 获取http响应内容，返回json
func (r *Result) Json(v interface{}) error {
	b, err := r.Raw()
	if err != nil {
		r.Err = err
		return r.Err
	}

	return json.Unmarshal(b, v)
}

// Save: 获取http响应内容，保存为文件
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
