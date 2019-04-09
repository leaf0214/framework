package sha

import (
	"crypto/sha256"
	"encoding/hex"
	b64 "xianhetian.com/framework/algorithm/base64"
)

//先求SHA256 哈希值，后使用base64编码
//返回base64 字符串
func Sha256(data string) string {
	return b64.Encode(SHA256Binary(data))
}

//SHA256 16进制哈希值
//返回16进制字符串
func SHA256Hex(data string) string {
	return hex.EncodeToString(SHA256Binary(data))
}

func SHA256BinaryHex(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

//SHA256 二进制数组哈希值
func SHA256Binary(data string) []byte {
	h := sha256.New()
	h.Write([]byte(data))
	return h.Sum(nil)
}
