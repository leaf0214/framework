package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"xianhetian.com/framework/algorithm/base64"
)

const (
	ModeCbc = "CBC"
	ModeCtr = "CTR"
)

type AES interface {
	EncryptStr(data string) (string, error)
	DecryptStr(data string) (string, error)
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
}

type CBC struct {
	block     cipher.Block
	blockSize int
	iv        []byte
}

func (cbc *CBC) EncryptStr(data string) (string, error) {
	encrypted, err := cbc.Encrypt([]byte(data))
	if err != nil {
		return "", nil
	}
	return base64.Encode(encrypted), nil
}

func (cbc *CBC) DecryptStr(data string) (string, error) {
	cryptByte, err := base64.Decode(data)
	if err != nil {
		return "", err
	}
	decrypted, err := cbc.Decrypt(cryptByte)
	return string(decrypted), nil
}

func (cbc *CBC) Encrypt(data []byte) ([]byte, error) {
	data = pKCS5Padding(data, cbc.blockSize)
	crypt := make([]byte, len(data))
	cipher.NewCBCEncrypter(cbc.block, cbc.iv).CryptBlocks(crypt, data)
	return crypt, nil
}

func (cbc *CBC) Decrypt(data []byte) ([]byte, error) {
	decrypt := make([]byte, len(data))
	cipher.NewCBCDecrypter(cbc.block, cbc.iv).CryptBlocks(decrypt, data)
	decrypt = pKCS5UnPadding(decrypt)
	return decrypt, nil
}

type CTR struct {
	block cipher.Block
	iv    []byte
}

func (ctr *CTR) EncryptStr(data string) (string, error) {
	encrypt, err := ctr.Encrypt([]byte(data))
	if err != nil {
		return "", nil
	}
	return base64.Encode(encrypt), nil
}

func (ctr *CTR) DecryptStr(data string) (string, error) {
	cryptByte, err := base64.Decode(data)
	if err != nil {
		return "", err
	}
	decrypted, err := ctr.Decrypt(cryptByte)
	return string(decrypted), nil
}

func (ctr *CTR) Encrypt(data []byte) ([]byte, error) {
	encrypt := make([]byte, len(data))
	cipher.NewCTR(ctr.block, ctr.iv).XORKeyStream(encrypt, data)
	return encrypt, nil
}

func (ctr *CTR) Decrypt(data []byte) ([]byte, error) {
	return ctr.Encrypt(data)
}

func NewAESStr(mode string, key string, iv string) (AES, error) {
	k, _ := hex.DecodeString(key)
	v, _ := hex.DecodeString(iv)
	return NewAES(mode, k, v)
}

func NewAES(mode string, k []byte, v []byte) (AES, error) {
	c, err := aes.NewCipher(k)
	if err != nil {
		return nil, fmt.Errorf("aes: mode %s, NewAES error %v", mode, err)
	}
	switch mode {
	case ModeCbc:
		return &CBC{
			block:     c,
			blockSize: c.BlockSize(),
			iv:        v,
		}, nil
	case ModeCtr:
		return &CTR{
			block: c,
			iv:    v,
		}, nil
	}
	return nil, fmt.Errorf("aes: unsupport mode: %v", mode)
}

func pKCS5Padding(text []byte, blockSize int) []byte {
	padding := blockSize - len(text)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(text, padText...)
}

func pKCS5UnPadding(data []byte) []byte {
	length := len(data)
	unPad := int(data[length-1])
	return data[:(length - unPad)]
}
