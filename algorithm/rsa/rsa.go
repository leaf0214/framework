package rsa

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	b64 "xianhetian.com/framework/algorithm/base64"
	cf "xianhetian.com/framework/config"
)

const (
	RSA_ALGORITHM_SIGN = crypto.SHA256
)

var length = cf.Config.DefaultInt("alg_ras_len", "1024")

//create RSA private key and public key
//return private key, public key
func GenerateKeyPair() (string, string) {
	key, err := rsa.GenerateKey(rand.Reader, length)
	if err != nil {
		return "", ""
	}
	pubKey := b64.Encode(x509.MarshalPKCS1PublicKey(&(key.PublicKey)))
	//获取公钥并base64编码
	//pkcs8私钥匙
	k, _ := x509.MarshalPKCS8PrivateKey(key)
	priKey := b64.Encode(k)
	return priKey, pubKey
}

//加密 方式:public key
//param:pubKey BASE64编码
//param:data  待加密数据
//return:string BASE64编码
func EncryptStr(pubKey string, data string) (string, error) {
	bytes, err := Encrypt(pubKey, []byte(data))
	if err != nil {
		return "", err
	}
	return b64.Encode(bytes), err
}

//解密 方式: private key
//param: privateKey, BASE64编码
//param: data, public key 加密后并BASE64编码的数据
//return: string
func DecryptStr(privateKey string, data string) (string, error) {
	byteData, _ := b64.Decode(data)
	bytes, err := Decrypt(privateKey, byteData)
	if err != nil {
		return "", err
	}
	return string(bytes), err
}

//加密 方式:public key
//param:pubKey BASE64编码
//param:data  待加密数据 []byte 类型
//return: []byte
func Encrypt(pubKey string, data []byte) ([]byte, error) {
	//1 get key from string
	key, err := getPubKey(pubKey)
	if err != nil {
		return nil, err
	}
	//2 encryption by RSA public key
	encryptData, err := rsa.EncryptPKCS1v15(rand.Reader, key, data)
	return encryptData, err
}

//解密 方式: private key
//param: priKey, BASE64编码
//param: data, public key 加密后的数据原始数据
//return: []byte
func Decrypt(priKey string, data []byte) ([]byte, error) {
	//1 get the rsa private key from string
	key, err := getPriKey(priKey)
	if err != nil {
		return nil, err
	}
	//2 decode by RSA private key
	decryptData, err := rsa.DecryptPKCS1v15(rand.Reader, key, data)
	return decryptData, err
}

//签名 方式: private key
//param: privateKey BASE64编码
//param: data 待签名数据
//return: string BASE64编码
func Sign(privateKey string, data string) (string, error) {
	//1 get the rsa private key from the privateKey string
	key, err := getPriKey(privateKey)
	if err != nil {
		return "", err
	}
	h := RSA_ALGORITHM_SIGN.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)
	encryptData, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed)
	if err != nil {
		return "", err
	}
	return b64.Encode(encryptData), nil
}

//认证 方式: public key
//param: publicKey BASE64编码
//param: data 原始签名数据
//param: encryptData 签名后BASE64编码的数据
//result err: 如果 err=nil 验证成功
func Verify(publicKey string, data string, encryptData string) error {
	//1 get key from string
	rsaPublicKey, err := getPubKey(publicKey)
	if err != nil {
		return err
	}
	encryptByte, _ := b64.Decode(encryptData)
	h := RSA_ALGORITHM_SIGN.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)
	//2 判断是否验证成功
	return rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed, encryptByte)
}

// 获取私钥
//param: priKey, private key 的字符串 BASE64编码
//result *rsa.PrivateKey
func getPriKey(priKey string) (*rsa.PrivateKey, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(priKey)
	if err != nil {
		return nil, err
	}
	t, err := x509.ParsePKCS8PrivateKey(privateKeyBytes)
	if err != nil {
		return nil, err
	}
	pri := t.(*rsa.PrivateKey)
	return pri, nil
}

// 获取公钥
//param: publicKey, public key 的字符串 BASE64编码
//result *rsa.PublicKey
func getPubKey(publicKey string) (*rsa.PublicKey, error) {
	pubbytes, err := b64.Decode(publicKey)
	if err != nil {
		return nil, err
	}
	pubInterface, err := x509.ParsePKIXPublicKey(pubbytes)
	if err != nil {
		return nil, err
	}
	pub := pubInterface.(*rsa.PublicKey)
	return pub, nil
}
