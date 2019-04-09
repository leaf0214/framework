package ecc

import (
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"math/big"
	"xianhetian.com/framework/math"
)

var (
	DefaultCurve                  = elliptic.P256()
	ErrInvalidCurve               = fmt.Errorf("ecies: invalid elliptic curve")
	ErrInvalidPublicKey           = fmt.Errorf("ecies: invalid public key")
	ErrSharedKeyIsPointAtInfinity = fmt.Errorf("ecies: shared key is point at infinity")
	ErrSharedKeyTooBig            = fmt.Errorf("ecies: shared key params are too big")
	ErrKeyDataTooLong             = fmt.Errorf("ecies: can't supply requested key data")
	ErrInvalidMessage             = fmt.Errorf("ecies: invalid message")
	big2To32                      = new(big.Int).Exp(big.NewInt(2), big.NewInt(32), nil)
	big2To32M1                    = new(big.Int).Sub(big2To32, big.NewInt(1))
	errInvalidPubKey              = errors.New("invalid  public key")
)

// PublicKey is a representation of an elliptic curve public key.
type PublicKey struct {
	X      *big.Int
	Y      *big.Int
	elliptic.Curve
	Params *ECIESParams
}

// PrivateKey is a representation of an elliptic curve private key.
type PrivateKey struct {
	PublicKey
	D *big.Int
}

// Generate an elliptic curve public / private keypair. If params is nil,
// the recommended default parameters for the key will be chosen.
func GenerateKey(rand io.Reader, curve elliptic.Curve, params *ECIESParams) (prv *PrivateKey, err error) {
	pb, x, y, err := elliptic.GenerateKey(curve, rand)
	if err != nil {
		return
	}
	prv = new(PrivateKey)
	prv.PublicKey.X = x
	prv.PublicKey.Y = y
	prv.PublicKey.Curve = curve
	prv.D = new(big.Int).SetBytes(pb)
	if params == nil {
		params = ParamsFromCurve(curve)
	}
	prv.PublicKey.Params = params
	return
}

// used to establish secret keys for encryption.
func (pri *PrivateKey) GenerateShared(pub *PublicKey, skLen int) (sk []byte, err error) {
	if pri.PublicKey.Curve != pub.Curve {
		return nil, ErrInvalidCurve
	}
	if skLen > MaxSharedKeyLength(pub) {
		return nil, ErrSharedKeyTooBig
	}
	x, _ := pub.Curve.ScalarMult(pub.X, pub.Y, pri.D.Bytes())
	if x == nil {
		return nil, ErrSharedKeyIsPointAtInfinity
	}
	sk = make([]byte, skLen)
	skBytes := x.Bytes()
	if len(skBytes) >= skLen {
		copy(sk, skBytes[:skLen])
	} else {
		copy(sk[len(sk)-len(skBytes):], skBytes)
	}
	return sk, nil
}

// 加密字符串
// pri 私钥 用于签名
// rand 产生随机数 IV向量
// mstr string 消息
// sstr  16进制字符 可通过交换产生，用于生成AES密钥的种子
func (pub *PublicKey) EncryptStr(pri *PrivateKey, rand io.Reader, mstr, sstr string) (ctStr string, err error) {
	m := []byte(mstr)
	var s1 []byte
	if sstr == "" {
		s1 = nil
	} else {
		s1, err = hex.DecodeString(sstr)
		if err != nil {
			return "", fmt.Errorf("sstr need hex code, error: %v", err)
		}
	}
	ct, err := pub.Encrypt(pri, rand, m, s1)
	if err != nil {
		return "", err
	}
	return string(ct), nil
}

// 字符串解密
func (pri *PrivateKey) DecryptStr(pub *PublicKey, cstr, sstr string) (string, error) {
	c := []byte(cstr)
	var s1 []byte
	if sstr == "" {
		s1 = nil
	} else {
		s1 = []byte(sstr)
	}
	msg, err := pri.Decrypt(pub, c, s1)
	if err != nil {
		return "", err
	}
	return string(msg), nil
}

// 加密
// pri 私钥 用于签名
// rand 产生随机数 IV向量
// m []byte 消息
// s1 []byte 可通过交换产生，用于生成AES密钥的种子
func (pub *PublicKey) Encrypt(pri *PrivateKey, rand io.Reader, m, s1 []byte) (ct []byte, err error) {
	params := pub.Params
	if params == nil {
		if params = ParamsFromCurve(pub.Curve); params == nil {
			err = ErrUnsupportedECIESParameters
			return
		}
	}
	R, err := GenerateKey(rand, pub.Curve, params)
	if err != nil {
		return
	}
	hash2 := params.Hash()
	skLen := params.KeyLen
	z, err := R.GenerateShared(pub, skLen)
	if err != nil {
		return
	}
	key, err := concatKDF(hash2, z, s1, skLen)
	if err != nil {
		return
	}
	em, err := symEncrypt(rand, params, key, m)
	if err != nil || len(em) <= params.BlockSize {
		return
	}
	//sign
	sLen := MaxSharedKeyLength(pub)
	Rb := elliptic.Marshal(pub.Curve, R.PublicKey.X, R.PublicKey.Y)
	ct = make([]byte, len(Rb)+len(em)+2*sLen)
	copy(ct, Rb)
	copy(ct[len(Rb):], em)
	r, s, err := pri.Sign(ct[:len(Rb)+len(em)])
	if err != nil {
		return nil, err
	}
	copy(ct[len(Rb)+len(em):], r.Bytes())
	copy(ct[len(Rb)+len(em)+sLen:], s.Bytes())
	return
}

// 解密
// pub 公钥 用于签名的验证
// c []byte 消息
// s1 []byte 可通过交换产生，用于生成AES密钥的种子
func (pri *PrivateKey) Decrypt(pub *PublicKey, c, s1 []byte) (m []byte, err error) {
	if len(c) == 0 {
		return nil, ErrInvalidMessage
	}
	params := pri.PublicKey.Params
	if params == nil {
		if params = ParamsFromCurve(pri.PublicKey.Curve); params == nil {
			err = ErrUnsupportedECIESParameters
			return
		}
	}
	hash2 := params.Hash()
	var (
		rLen   int
		mStart int
		mEnd   int
	)
	sLen := MaxSharedKeyLength(pub)
	switch c[0] {
	case 2, 3, 4:
		rLen = (pri.PublicKey.Curve.Params().BitSize + 7) / 4
		if len(c) < (rLen + pub.Params.BlockSize + 2*sLen) {
			err = ErrInvalidMessage
			return
		}
	default:
		err = ErrInvalidPublicKey
		return
	}
	mStart = rLen
	mEnd = len(c) - 2*sLen
	R := new(PublicKey)
	R.Curve = pri.PublicKey.Curve
	R.X, R.Y = elliptic.Unmarshal(R.Curve, c[:rLen])
	if R.X == nil {
		err = ErrInvalidPublicKey
		return
	}
	if !R.Curve.IsOnCurve(R.X, R.Y) {
		err = ErrInvalidCurve
		return
	}
	// verify
	s := c[mEnd:]
	r := s[:sLen]
	s = s[sLen:]
	if ok := pub.Verify(c[:mEnd], new(big.Int).SetBytes(r), new(big.Int).SetBytes(s)); !ok {
		err = ErrInvalidMessage
		return
	}
	skLen := params.KeyLen
	z, err := pri.GenerateShared(R, skLen)
	if err != nil {
		return
	}
	key, err := concatKDF(hash2, z, s1, skLen)
	if err != nil {
		return
	}
	m, err = symDecrypt(params, key, c[mStart:mEnd])
	return
}

//签名
//hash 消息的hash值
func (pri *PrivateKey) Sign(msg []byte) (r, s *big.Int, err error) {
	hash := pri.Params.Hash()
	hash.Write(msg)
	hashVal := hash.Sum(nil)
	hash.Reset()
	priKeyEcd := pri.toECDSA()
	zero := big.NewInt(0)
	r, s, err = ecdsa.Sign(rand.Reader, priKeyEcd, hashVal)
	if err != nil {
		return zero, zero, err
	}
	return r, s, nil
}

func (pri *PrivateKey) SignHex(msg []byte) (signature string, err error) {
	sig, err := pri.SignByte(msg)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sig), nil
}

func (pri *PrivateKey) SignByte(msg []byte) (signature []byte, err error) {
	r, s, err := pri.Sign(msg)
	if err != nil {
		return nil, err
	}
	sig := append(r.Bytes(), s.Bytes()...)
	return sig, nil
}

//验证
//hash 消息hash值
func (pub *PublicKey) Verify(msg []byte, r *big.Int, s *big.Int) (result bool) {
	pubKeyEcd := pub.toECDSA()
	hash := pub.Params.Hash()
	hash.Write(msg)
	hashVal := hash.Sum(nil)
	hash.Reset()
	return ecdsa.Verify(pubKeyEcd, hashVal, r, s)
}

func (pub *PublicKey) VerifyHex(msg []byte, sig string) (result bool) {
	pubKeyEcd := pub.toECDSA()
	size := (pubKeyEcd.Params().BitSize + 7) / 8
	bytes, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	r := new(big.Int).SetBytes(bytes[:size])
	s := new(big.Int).SetBytes(bytes[size: 2*size])
	hash := pub.Params.Hash()
	hash.Write(msg)
	hashVal := hash.Sum(nil)
	hash.Reset()
	return ecdsa.Verify(pubKeyEcd, hashVal, r, s)
}

func (pub *PublicKey) VerifyByte(msg []byte, sig []byte) (result bool) {
	pubKeyEcd := pub.toECDSA()
	size := (pubKeyEcd.Params().BitSize + 7) / 8
	r := new(big.Int).SetBytes(sig[:size])
	s := new(big.Int).SetBytes(sig[size: 2*size])
	hash := pub.Params.Hash()
	hash.Write(msg)
	hashVal := hash.Sum(nil)
	hash.Reset()
	return ecdsa.Verify(pubKeyEcd, hashVal, r, s)
}

//公钥导出16进制字符串
func (pub *PublicKey) Export() string {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return ""
	}
	return hex.EncodeToString(elliptic.Marshal(pub.Curve, pub.X, pub.Y))
}

//私钥导出16进制字符串
func (pri *PrivateKey) Export() string {
	if pri == nil {
		return ""
	}
	return hex.EncodeToString(math.PaddedBigBytes(pri.D, pri.Curve.Params().BitSize/8))
}

//16进制的公钥字符串转换为公钥
func ImportPubHex(curve elliptic.Curve, dhex string) (*PublicKey, error) {
	keyByte, err := hex.DecodeString(dhex)
	if err != nil {
		return nil, fmt.Errorf("invalid public key, decode hex error: %v", err)
	}
	return ImportPub(curve, keyByte)
}

func ImportPub(curve elliptic.Curve, keyByte []byte) (*PublicKey, error) {
	x, y := elliptic.Unmarshal(curve, keyByte)
	if x == nil {
		return nil, errInvalidPubKey
	}
	return &PublicKey{
		X:      x,
		Y:      y,
		Curve:  curve,
		Params: ParamsFromCurve(curve),
	}, nil
}

//16进制的私钥字符串转换为私钥
func ImportPriHex(curve elliptic.Curve, dhex string) (*PrivateKey, error) {
	keyByte, err := hex.DecodeString(dhex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key, decode hex error: %v", err)
	}
	return ImportPri(curve, keyByte)
}

func ImportPri(curve elliptic.Curve, keyByte []byte) (*PrivateKey, error) {
	pri := new(PrivateKey)
	pri.PublicKey.Curve = curve
	if len(keyByte) != (pri.Curve.Params().BitSize+7)/8 {
		return nil, fmt.Errorf("invalid length, need %d bits", pri.Curve.Params().BitSize)
	}
	pri.D = new(big.Int).SetBytes(keyByte)
	// The pri.D must not be zero or negative.
	if pri.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}
	pri.X, pri.PublicKey.Y = pri.PublicKey.Curve.ScalarBaseMult(keyByte)
	if pri.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	params := ParamsFromCurve(curve)
	pri.PublicKey.Params = params
	return pri, nil
}

//切换到库中的ECDSA 公钥
func (pub *PublicKey) toECDSA() *ecdsa.PublicKey {
	return &ecdsa.PublicKey{Curve: pub.Curve, X: pub.X, Y: pub.Y}
}

//切换到库中的ECDSA 私钥
func (pri *PrivateKey) toECDSA() *ecdsa.PrivateKey {
	return &ecdsa.PrivateKey{
		PublicKey: *(pri.PublicKey.toECDSA()),
		D:         pri.D,
	}
}

func KeyString(key []byte) string {
	return fmt.Sprintf("%X", key[:])
}

//MaxSharedKeyLength returns the maximum length of the shared key the
//public key can produce.
func MaxSharedKeyLength(pub *PublicKey) int {
	return (pub.Curve.Params().BitSize + 7) / 8
}

// Generate an initialisation vector for CTR mode.
func generateIV(params *ECIESParams, rand io.Reader) (iv []byte, err error) {
	iv = make([]byte, params.BlockSize)
	_, err = io.ReadFull(rand, iv)
	return
}

// symDecrypt carries out CTR decryption using the block cipher specified in
// the parameters
func symDecrypt(params *ECIESParams, key, ct []byte) (m []byte, err error) {
	c, err := params.Cipher(key)
	if err != nil {
		return
	}
	ctr := cipher.NewCTR(c, ct[:params.BlockSize])
	m = make([]byte, len(ct)-params.BlockSize)
	ctr.XORKeyStream(m, ct[params.BlockSize:])
	return
}

func incCounter(ctr []byte) {
	if ctr[3]++; ctr[3] != 0 {
		return
	}
	if ctr[2]++; ctr[2] != 0 {
		return
	}
	if ctr[1]++; ctr[1] != 0 {
		return
	}
	if ctr[0]++; ctr[0] != 0 {
		return
	}
}

// symEncrypt carries out CTR encryption using the block cipher specified in the
// parameters.
func symEncrypt(rand io.Reader, params *ECIESParams, key, m []byte) (ct []byte, err error) {
	c, err := params.Cipher(key)
	if err != nil {
		return
	}
	iv, err := generateIV(params, rand)
	if err != nil {
		return
	}
	ctr := cipher.NewCTR(c, iv)
	ct = make([]byte, len(m)+params.BlockSize)
	copy(ct, iv)
	ctr.XORKeyStream(ct[params.BlockSize:], m)
	return
}

// NIST Concatenation Key Derivation Function (see section).
func concatKDF(hash hash.Hash, z, s1 []byte, kdLen int) (k []byte, err error) {
	if s1 == nil {
		s1 = make([]byte, 0)
	}
	reps := ((kdLen + 7) * 8) / (hash.BlockSize() * 8)
	if big.NewInt(int64(reps)).Cmp(big2To32M1) > 0 {
		fmt.Println(big2To32M1)
		return nil, ErrKeyDataTooLong
	}
	counter := []byte{0, 0, 0, 1}
	k = make([]byte, 0)
	for i := 0; i <= reps; i++ {
		hash.Write(counter)
		hash.Write(z)
		hash.Write(s1)
		k = append(k, hash.Sum(nil)...)
		hash.Reset()
		incCounter(counter)
	}
	k = k[:kdLen]
	return
}
