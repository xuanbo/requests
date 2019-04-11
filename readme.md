# requests

> http requests lib for golang

## Features

* `GET`、`POST`、`PUT`、`DELETE`
* `application/json`、`application/x-www-form-urlencoded`

## Todo

* `multipart/form-data`

## Examples

### Get

```go
func getText() {
	text, err := requests.Get("http://127.0.0.1:8080/ping").
		Params(url.Values{
			"param1": {"value1"},
			"param2": {"123"},
		}).
		Send().
		Text()
	if err != nil {
		panic(err)
	}
	fmt.Println(text)
}
```

query

```
GET http://127.0.0.1:8080/ping?param1=value1&param2=123 HTTP/1.1
```

### Post Form

```go
func postForm() {
	text, err := requests.Post("http://127.0.0.1:8080/ping").
		Params(url.Values{
			"param1": {"value1"},
			"param2": {"123"},
		}).
		Form(url.Values{
			"form1": {"value1"},
			"form2": {"123"},
		}).
		Send().
		Text()
	if err != nil {
		panic(err)
	}
	fmt.Println(text)
}
```

post form

```
POST http://127.0.0.1:8080/ping?param1=value1&param2=123 HTTP/1.1
Content-Type: application/x-www-form-urlencoded

form1=value1&form2=123
```

### Post Json

```go
func postJson() {
	text, err := requests.Post("http://127.0.0.1:8080/ping").
		Params(url.Values{
			"param1": {"value1"},
			"param2": {"123"},
		}).
		Json(map[string]interface{}{
			"json1": "value1",
			"json2": 2,
		}).
		Send().
		Text()
	if err != nil {
		panic(err)
	}
	fmt.Println(text)
}
```

post json

```
POST http://127.0.0.1:8080/ping?param1=value1&param2=123 HTTP/1.1
Content-Type: application/json

{"json1": "value1", "json2": 2}
```

### Save File

```go
func save() {
	err := requests.Get("https://github.com/xuanbo/requests").
		Send().
		Save("./requests.html")
	if err != nil {
		panic(err)
	}
}
```
