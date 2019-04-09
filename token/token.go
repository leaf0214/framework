package token

import (
	"encoding/json"
	"errors"
	"time"
	b64 "xianhetian.com/framework/algorithm/base64"
	"xianhetian.com/framework/algorithm/rsa"
	"xianhetian.com/framework/algorithm/sha"
	cf "xianhetian.com/framework/config"
)

const (
	RSA   = "RSA"
	TOKEN = "TOKEN"
)

var (
	pubKey  = cf.Config.DefaultString("sec_pub_key", "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCOOPl/m02msH5ORJ2IgWQcW6T466vc9KhRiWvnvS8Ksk2iIWOubVbRJujA3LTp1XGPdm///oowpPTXh4eFW6dg7ymcQWZCM1FJe3i9sC2eOqueUGcXgizAp9oUnzXuYpWgG+yYFvQiUTapMYQZxT+YA4efSzenxf5LzohoE+i+HwIDAQAB")
	priKey  = cf.Config.DefaultString("sec_pri_key", "MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBAI44+X+bTaawfk5EnYiBZBxbpPjrq9z0qFGJa+e9LwqyTaIhY65tVtEm6MDctOnVcY92b//+ijCk9NeHh4Vbp2DvKZxBZkIzUUl7eL2wLZ46q55QZxeCLMCn2hSfNe5ilaAb7JgW9CJRNqkxhBnFP5gDh59LN6fF/kvOiGgT6L4fAgMBAAECgYAO2Z4bl+C8xfL6Qynbxf7pAxyvrRPt51Hn6ZxtvxA5YrK+ehQJc3s8LX7iHGl7fQD1hN1e8noFaEP0eT9KSm6ohV4EKRVLLgaiAdEgLcoAkAiWpztphUxr1vUTrqfCw8OZ/OBtrtBIqnHPFl44e83vcCB0phZSWeDNglfKnB1lgQJBANQpmcD2A7Ex//Ao8MA/3qUHFWfFcFn34/1NFOfC66N6Th5f/0/wSzCmIdGdsVcNkdsdmin167htnV59VtTyrd8CQQCrm+EqTXFMvpR16G99LxdiuKxhbW/pnvP1cx/TJ2Nu2PadubFia1/o/gJpPbrFHyj8FjEHjo8wuas3BLWX/3fBAkAbZ1gtvVkSvSOS0KbwHg/S/ww7wBvX8xXmtNsbaGjpT7XhZILkv2Pm376EhbrPRLhvNe6gttwAkV//QW9CyCm/AkBtwS16O7t55O3Yl0cu3j5rskb1rOOFnFbVJcM17hwnGfZonAn6M0hNIJ/0JTndtvckeyDyf1fPRwBdGNL3mrlBAkBX3Q5ELpYOI91PpCppsv19LyvbBQSc7bsSivKemg/lSaocrDYUmSCya/wHI9o7Y766ZpwLrr1ctQnEonJZHmD2")
	timeout = int64(cf.Config.DefaultInt("token_timeout", "3600"))
)

type Token struct {
	Header Header    // 头信息
	Body   Body      // 传输数据
	Sign   Signature // 签名
}

type Header struct {
	Type string // Token类型，JWT不加密Body，JWTS加密Body
	ALG  string // 加密签名的算法
}

type Body struct {
	Id        string      `json:"id,omitempty"`        // 发布Token用户或唯一标识
	IP        string      `json:"ip,omitempty"`        // 客户端IP
	Timestamp int64       `json:"timestamp,omitempty"` // 发布时间戳
	Timeout   int64       `json:"timeout,omitempty"`   // 过期时间
	Data      interface{} `json:"data,omitempty"`      // 扩展数据
}

type Signature struct {
	Hash  string `json:"hash,omitempty"`      // Header与Body的哈希值
	Value string `json:"signature,omitempty"` // 签名
}

/*
创建一个新Token
传入参数：data(包含数据的Body结构体）
返回结果：string(Token字符串)；error(错误)
*/
func NewToken(data Body) (string, error) {
	if data.Timeout == 0 {
		data.Timeout = time.Now().Unix() + timeout
	}
	body, _ := json.Marshal(data)
	hash := sha.Sha256(string(body[:]))
	sign, err := rsa.Sign(priKey, hash)
	if err != nil {
		return "", err
	}
	token, _ := json.Marshal(&Token{Header: Header{Type: TOKEN, ALG: RSA}, Body: data, Sign: Signature{Hash: hash, Value: sign}})
	return b64.URLEncode(token), nil
}

/*
验证Token
传入参数：需要验证的Token字符串
返回结果：success(是否验证成功)；body（数据）
*/
func Verify(str string) (success bool, body *Body) {
	success = false
	b, _ := b64.URLDecode(str)
	token := Token{}
	if json.Unmarshal(b, &token) != nil {
		return
	}
	m, _ := json.Marshal(token.Body)
	hash := sha.Sha256(string(m[:]))
	if hash != token.Sign.Hash {
		return
	}
	if token.Body.valid() != nil {
		return
	}
	if rsa.Verify(pubKey, hash, token.Sign.Value) != nil {
		return
	}
	return true, &token.Body
}

/*
校验Body的过期时间是否过期
*/
func (body Body) valid() error {
	now := time.Now().Unix()
	if !(now <= body.Timeout) {
		delta := time.Unix(now, 0).Sub(time.Unix(body.Timeout, 0))
		return errors.New("the token expired: " + delta.String())
	}
	return nil
}
