package random

import (
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"math"
	"math/rand"
	"strconv"
	"time"
	"xianhetian.com/framework/algorithm/sha"
)

var num uint32 = 0

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

//定义随机数数组 前52为非数字 后10位 0到9的数字
var letters = [62]string{4: "P", 8: "Q", 11: "R", 14: "S", 15: "T", 19: "U", 22: "N", 33: "O", 42: "V", 2: "W", 5: "X", 32: "Y", 43: "Z", 3: "a", 30: "b", 34: "c", 51: "d", 1: "f", 25: "g", 31: "h", 37: "i", 44: "j", 46: "k", 47: "e", 10: "l", 17: "m", 23: "n", 49: "o", 20: "q", 21: "r", 26: "s", 27: "t", 36: "u", 39: "v", 50: "p", 6: "y", 7: "z", 12: "A", 13: "B", 18: "C", 24: "D", 28: "w", 35: "x", 40: "E", 48: "F", 0: "H", 9: "I", 16: "J", 29: "K", 38: "L", 41: "M", 45: "G", 53: "8", 56: "9", 57: "3", 58: "4", 59: "5", 60: "6", 52: "7", 55: "1", 61: "2", 54: "0"}

//存放当前随机数的值
var randMap map[string][2]int = make(map[string][2]int)

func ValStr(lens int) string {
	return hex.EncodeToString(Value(lens))
}

func Value(lens int) []byte {
	b := make([]byte, lens)
	rand.Read(b)
	return b
}

//获取随机数
//默认返回44位随机数
func UqValue() string {
	//1 按时间获取6位随机数
	st := strconv.FormatInt(time.Now().Unix(), 10)
	//2 获取字符串随机数
	b := make([]byte, 6)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	//3 拼接字符串
	st = st + string(b)
	//4 求hashCode
	st = sha.Sha256(st)
	//5 加上固定值
	st = st + increaseNum()
	return sha.Sha256(st)
}

//获取随机数
//按长度返回随机数,最大可以返回88位
func RanValLen(lens int8) string {
	//1 按时间获取6位随机数
	st := strconv.FormatInt(time.Now().Unix(), 10)
	//2 获取字符串随机数
	b := make([]byte, 6)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	//3 拼接字符串
	st = st + string(b)
	//4 求hashCode
	s2 := sha.Sha256(st)
	//5 加上固定值
	st = s2 + increaseNum()
	st = sha.Sha256(st)
	switch {
	case lens <= 44:
		return st[:lens]
	default:
		return (st + s2)[:lens]
	}
}

func increaseNum() string {
	//循环使用
	if num >= math.MaxUint32 {
		num = 0
	}
	num++
	return fmt.Sprint(num)
}

//random value parameter
//通过传入的标志返回随机数
//最少6位
//@parameter par 当前随机数名字,par 使用默认
//@parameter lenth 长度
func RanValPar(par string, lenth int) (string, error) {
	//处理逻辑 YMDH+2位顺序数字+(len-6)位随机数
	var year = 2018 //开始年份
	lens := len(letters)
	//返回结果
	var result string
	if lenth < 6 {
		return "", errors.New("位数少于6")
	}

	//1 生成1到10的随机数
	ran := rand.Intn(10)

	//2  取模 月份模 12*5<60 添数 31*2-1 <62
	mm := ran % 5
	md := ran % 2

	//3 获取当前年月日时
	t := time.Now()
	y := t.Year()
	m := int(t.Month())
	d := t.Day()
	h := t.Hour()
	min := t.Minute()

	//4 年月日时组装
	result = letters[y-year] + letters[m+12*mm] + letters[d+31*md-1] + letters[h+24*md] + letters[min]

	//5 获取顺序值
	if par == "" {
		par = "def"
	}
	v, ok := randMap[par]
	if !ok {
		randMap[par] = [2]int{0, 0}
		v = randMap[par]
	}
	if v[1] >= (lens - 1) {
		v[1] = 0
		if v[0] >= (lens - 1) {
			v[0] = 0
		} else {
			v[0]++
		}
	} else {
		v[1]++
	}
	randMap[par] = v

	//6 年月日时分+固定值组装
	result += letters[v[0]] + letters[v[1]]

	//7年月日时+固定值组装+(lenth-7)位随机数
	for i := 0; i < (lenth - 7); i++ {
		result += letters[rand.Intn(lens)]
	}
	if lenth == 6 {
		result = result[0:6]
	}
	return result, nil
}
