package zip

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"xianhetian.com/framework/algorithm/base64"
)

// 压缩数据并转换为BASE64编码字符串
func Compress(str string) string {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	defer w.Close()
	w.Write([]byte(str))
	w.Flush()
	bstr := base64.Encode(b.Bytes())
	return bstr
}

// 解压BASE64编码的压缩数据
func Decompress(bstr string) string {
	str, _ := base64.DecodeStr(bstr)
	var b bytes.Buffer
	b.Write([]byte(str))
	r, _ := gzip.NewReader(&b)
	defer r.Close()
	d, _ := ioutil.ReadAll(r)
	return string(d[:])
}
