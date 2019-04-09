package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	cf "xianhetian.com/framework/config"
)

var (
	ht   = cf.Config.DefaultInt("http_timeout", "10000")                // 超时时间
	htc  = cf.Config.DefaultInt("http_retry_count", "3")                // 重试次数
	hit  = cf.Config.DefaultInt("http_interval_time", "0")              // 间隔时间
	conn = cf.Config.DefaultString("http_connection", "close")          // 设置close则为短连接每一次请求都关闭链接，设置keep-alive则为长连接
	ka   = cf.Config.DefaultString("http_keepalive", "60")              // 设置长连接过期时间
	ce   = cf.Config.DefaultString("http_content_encoding", "identity") // 压缩模式,设置后浏览器自动处理解压，默认identity
)

type httpClient struct {
	client       *http.Client
	retryCount   int           // 重试次数
	intervalTime time.Duration // 间隔时间
}

type Request struct {
	Url    string            // 传输URL
	Method string            // 请求方法
	Data   interface{}       // 传输数据
	Header map[string]string // HTTP请求头
}

// 返回一个HTTPClient的实例
func NewHTTPClient() *httpClient {
	return &httpClient{
		retryCount:   htc,
		intervalTime: time.Duration(hit) * time.Millisecond,
		client:       &http.Client{Timeout: time.Duration(ht) * time.Millisecond},
	}
}

// GET请求方法
func (h *httpClient) Get(r *Request) (string, error) {
	r.setMethod(http.MethodGet)
	return h.service(r)
}

// POST请求方法
func (h *httpClient) Post(r *Request) (string, error) {
	r.setMethod(http.MethodPost)
	return h.service(r)
}

// 设置Request的URL
func (r *Request) SetUrl(url string) {
	r.Url = url
}

// 设置Request的数据
func (r *Request) SetData(data interface{}) {
	r.Data = data
}

// 设置Request自定义的请求头
func (r *Request) SetHeader(header map[string]string) {
	r.Header = header
}

// 设置请求方法
func (r *Request) setMethod(method string) {
	r.Method = method
}

func (h *httpClient) service(r *Request) (string, error) {
	var str string
	switch r.Data.(type) {
	case string:
		d, ok := r.Data.(string)
		if ok {
			str = d
		}
	case map[string]string:
		m, _ := r.Data.(map[string]string)
		var param bytes.Buffer
		for k, v := range m {
			param.WriteString(fmt.Sprintf("&%s=%s", k, v))
		}
		rs := []rune(param.String())
		str = string(rs[1:])
	}
	req, _ := http.NewRequest(r.Method, r.Url, strings.NewReader(str))
	if len(r.Header) > 0 {
		for k, v := range r.Header {
			req.Header.Add(k, v)
		}
	}
	req.Header.Add("Connection", conn)
	req.Header.Add("Keep-Alive", ka)
	req.Header.Add("Content-Encoding", ce)
	res, err := h.send(req)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(res.Body)
	return string(body[:]), nil
}

func (h *httpClient) send(request *http.Request) (response *http.Response, err error) {
	for i := 0; i <= h.retryCount; i++ {
		response, err = h.client.Do(request)
		if err != nil || response.StatusCode >= http.StatusInternalServerError {
			time.Sleep(h.intervalTime)
			continue
		}
		// 如果重试后成功传输，则将重试过程中的err置为nil
		err = nil
		break
	}
	return
}
