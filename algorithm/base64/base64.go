package base64

import "encoding/base64"

//base64编码
//参数：二进制数组
func Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

//base64解码
//参数：string
//结果：二进制数组
func Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

//base64编码
//参数：字符串
func EncodeStr(data string) string {
	return Encode([]byte(data))
}

//base64编码
//参数：字符串
//结果：字符串
func DecodeStr(data string) (string, error) {
	decodeBytes, err := Decode(data)
	return string(decodeBytes), err
}

//base64编码
//参数：字符串
//结果：字符串
func URLEncode(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

//base64编码
//参数：字符串
//结果：二进制数组
func URLDecode(data string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(data)
}
