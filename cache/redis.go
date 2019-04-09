package cache

import (
	"encoding/json"
	"github.com/garyburd/redigo/redis"
	"time"
	"xianhetian.com/framework/logger"
)

var Pool *redis.Pool

type RedisConfig struct {
	Addr           string // RedisIP地址
	Password       string // 密码
	DbNum          int    // 数据库编号
	MaxIdle        int    // 最大空闲连接数
	ReadTimeout    int64  // 读取超时时间；单位：毫秒
	WriteTimeout   int64  // 写入超时时间；单位：毫秒
	IdleTimeout    int64  // 空闲超时时间；单位：毫秒
	ConnectTimeout int64  // 连接超时时间；单位：秒
}

/*
设置缓存
Set("key", str) 设置String缓存
Set("key", str, 5000) 设置String缓存并将缓存过期时间设置为5000毫秒
Set("key", []string)  将数组以Redis的List类型设置为缓存
Set("key", map[string]string) 将Map以Redis的Hash类型设置为key-value缓存
*/
func Set(key string, val interface{}, expire ...int) (i interface{}, err error) {
	if err = ping(); err != nil {
		return
	}
	var value interface{}
	switch v := val.(type) {
	case string, int, uint, int8, int16, int32, int64, float32, float64, bool:
		value = v
	case []string:
		n, _ := val.([]string)
		for _, v := range n {
			i, err = Do("LPUSH", key, v)
			logInf(err, key, i)
			if len(expire) <= 0 {
				i, err = Do("EXPIRE", key, 600000)
				logInf(err, key, i)
			} else {
				i, err = Do("EXPIRE", key, expire[0])
				logInf(err, key, i)
			}
		}
		return
	case map[string]string:
		m, _ := val.(map[string]string)
		for k, v := range m {
			Do("HSET", key, k, v)
			if len(expire) <= 0 {
				i, err = Do("EXPIRE", key, 600000)
				logInf(err, key, i)
				return
			}
			i, err = Do("EXPIRE", key, expire[0])
			logInf(err, key, i)
			return
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		value = string(b[:])
	}
	if len(expire) <= 0 {
		logger.Debug(value)
		i, err = Do("SETEX", key, 6000, value)
		logInf(err, key, i)
		return
	}
	logger.Debug(value)
	i, err = Do("SETEX", key, expire[0], value)
	logInf(err, key, i)
	return
}

/*
根据key获取缓存
Get("key", str) 获取缓存
Get("key", str, hashKey string) 根据哈希Key值获取缓存
Get("key", str, listIndex int) 根据List坐标获取String缓存
*/
func Get(key string, param ...interface{}) (i interface{}, err error) {
	if err = ping(); err != nil {
		return
	}
	if len(param) <= 0 {
		i, err = Do("GET", key)
		logInf(err, key, i)
		return
	}
	for _, p := range param {
		switch p.(type) {
		case string:
			i, err = Do("HGET", key, p)
			logInf(err, key, i)
			return
		case int:
			i, err = redis.String(Do("LINDEX", key, p))
			logInf(err, key, i)
			return
		}
	}
	return
}

/*
根据key获取string类型缓存
GetStr("key", str) 获取string类型缓存
GetStr("key", str, hashKey string) 根据哈希Key值获取string类型缓存
*/
func GetStr(key string, field ...string) (s string, err error) {
	if err = ping(); err != nil {
		return
	}
	if len(field) > 0 {
		s, err = redis.String(Do("HGET", key, field))
		logInf(err, key, s)
		return
	}
	s, err = redis.String(Do("GET", key))
	logInf(err, key, s)
	return
}

/*
根据key获取Int类型缓存
GetInt("key", str) 获取Int类型缓存
GetInt("key", str, hashKey string) 根据哈希Key值获取Int类型缓存
*/
func GetInt(key string, field ...string) (i int, err error) {
	if err := ping(); err != nil {
		return 0, err
	}
	if len(field) > 0 {
		return redis.Int(Do("HGET", key, field))
	}
	i, err = redis.Int(Do("GET", key))
	logInf(err, key, i)
	if err != nil {
		return 0, err
	}
	return
}

/*
根据key获取Int64类型缓存
GetInt64("key", str) 获取Int64类型缓存
GetInt64("key", str, hashKey string) 根据哈希Key值获取Int64类型缓存
*/
func GetInt64(key string, field ...string) (i int64, err error) {
	if err := ping(); err != nil {
		return 0, err
	}
	if len(field) > 0 {
		return redis.Int64(Do("HGET", key, field))
	}
	i, err = redis.Int64(Do("GET", key))
	logInf(err, key, i)
	if err != nil {
		return 0, err
	}
	return
}

/*
根据key获取Bool类型缓存
GetBool("key", str) 获取Bool类型缓存
GetBool("key", str, hashKey string) 根据哈希Key值获取Bool类型缓存
*/
func GetBool(key string, field ...string) (b bool, err error) {
	if err := ping(); err != nil {
		return false, err
	}
	if len(field) > 0 {
		return redis.Bool(Do("HGET", key, field))
	}
	b, err = redis.Bool(Do("GET", key))
	logInf(err, key, b)
	if err != nil {
		return false, err
	}
	return
}

/*
根据key获取Struct类型缓存
GetStruct("key", str) 获取Struct类型缓存
GetStruct("key", str, hashKey string) 根据哈希Key值获取Struct类型缓存
*/
func GetStruct(key string, val interface{}, field ...string) (err error) {
	if err := ping(); err != nil {
		return err
	}
	var r string
	if len(field) > 0 {
		r, err = GetStr(key, field[0])
	} else {
		r, err = GetStr(key)
	}
	if err != nil {
		return
	}
	u := json.Unmarshal([]byte(r), val)
	logInf(err, key, u)
	return
}

/*
根据key获取所有的域和值
GetAll("key", val interface{}) 根据哈希Key值获取所有的域和值
*/
func GetAll(key string) (v interface{}, err error) {
	if err = ping(); err != nil {
		return
	}
	v, err = redis.Values(Do("HGETALL", key))
	logger.Debug(v)
	if err != nil {
		return
	}
	return
}

// 检查键是否存在
func Exists(key string) (b bool, err error) {
	if err := ping(); err != nil {
		return false, err
	}
	b, err = redis.Bool(Do("EXISTS", key))
	logInf(err, key, b)
	if err != nil {
		return false, err
	}
	return
}

// 删除键
func Del(key string) (err error) {
	if err = ping(); err != nil {
		return err
	}
	r, err := Do("DEL", key)
	logInf(err, key, r)
	return
}

// 将key中储存的数字值增一
func Incr(key string) (i int64, err error) {
	if err := ping(); err != nil {
		return 0, err
	}
	i, err = redis.Int64(Do("INCR", key))
	logInf(err, key, i)
	if err != nil {
		return 0, err
	}
	return
}

// 将key所储存的值加上增量increment
func IncrBy(key string, amount int) (i int64, err error) {
	if err := ping(); err != nil {
		return 0, err
	}
	i, err = redis.Int64(Do("INCRBY", key, amount))
	logInf(err, key, i)
	if err != nil {
		return 0, err
	}
	return
}

// 将key中储存的数字值减一
func Decr(key string) (i int64, err error) {
	if err := ping(); err != nil {
		return 0, err
	}
	i, err = redis.Int64(Do("DECR", key))
	logInf(err, key, i)
	if err != nil {
		return 0, err
	}
	return
}

// 将key所储存的值减去减量decrement
func DecrBy(key string, amount int) (i int64, err error) {
	if err := ping(); err != nil {
		return 0, err
	}
	i, err = redis.Int64(Do("DECRBY", key, amount))
	logInf(err, key, i)
	if err != nil {
		return 0, err
	}
	return
}

func Do(commandName string, args ...interface{}) (interface{}, error) {
	conn := Pool.Get()
	defer conn.Close()
	return conn.Do(commandName, args...)
}

func Send(commandName string, args ...interface{}) error {
	conn := Pool.Get()
	defer conn.Close()
	return conn.Send(commandName, args...)
}

func ping() (err error) {
	if _, err = Do("PING"); err != nil {
		logger.Error("Redis PING 失败 , Err：%s", err)
		panic(err)
	}
	return
}

func logInf(err error, key string, result interface{}) {
	if err == nil {
		logger.Info("Redis信息： Key = %s , Result：%s", key, result)
		return
	}
	logger.Error("Redis信息： Err = %s , Key = %s", err, key)
}

func NewPool(rc *RedisConfig) *redis.Pool {
	return &redis.Pool{
		MaxActive:   rc.MaxIdle,
		MaxIdle:     rc.MaxIdle,
		IdleTimeout: time.Duration(rc.IdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", rc.Addr,
				redis.DialConnectTimeout(time.Millisecond*time.Duration(rc.ConnectTimeout)),
				redis.DialReadTimeout(time.Millisecond*time.Duration(rc.ReadTimeout)),
				redis.DialWriteTimeout(time.Millisecond*time.Duration(rc.WriteTimeout)))
			if err != nil {
				logger.Error(err)
				return nil, err
			}
			if rc.Password != "" {
				if _, err = c.Do("AUTH", rc.Password); err != nil {
					logger.Error(err)
					return nil, err
				}
			}
			if err = c.Send("SELECT", rc.DbNum); err != nil {
				logger.Error(err)
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}
