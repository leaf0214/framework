package config

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DirName    = "config"
	FileName   = "framework.conf"
	VendorPath = "/vendor/xianhetian.com/framework"
)

var Config properties

type part struct {
	vals map[string]string
}

type properties struct {
	values map[string]string
}

func init() {
	var err error
	var appConfigPath string
	appPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	appConfigPath = filepath.Join(appPath, DirName, FileName)
	if !fileExists(appConfigPath) {
		appPath += VendorPath
		appConfigPath = filepath.Join(appPath, DirName, FileName)
		if !fileExists(appConfigPath) {
			return
		}
	}
	Init(appConfigPath)
}

func (p *properties) Set(k, v string) {
	Config.values[k] = v
}

func (p *properties) Get(key string) string {
	v, _ := p.values[key]
	return v
}

func (p *properties) String(key string) string {
	return p.Get(key)
}

func (p *properties) DefaultString(key string, defaultVal string) string {
	if v := p.Get(key); v != "" {
		return v
	}
	return defaultVal
}

func (p *properties) Int(key string) int {
	v := p.Get(key)
	v2, _ := strconv.Atoi(v)
	return v2
}

func (p *properties) DefaultInt(key string, defaultVal string) int {
	result := defaultVal
	if v := p.Get(key); v != "" {
		result = v
	}
	v2, _ := strconv.Atoi(result)
	return v2
}

func (p *properties) Float(key string) float64 {
	v := p.Get(key)
	v2, _ := strconv.ParseFloat(v, 10)
	return v2
}

func (p *properties) DefaultFloat(key string, defaultVal string) float64 {
	result := defaultVal
	if v := p.Get(key); v != "" {
		result = v
	}
	v2, _ := strconv.ParseFloat(result, 10)
	return v2
}

func (p *properties) Bool(key string) bool {
	v := p.Get(key)
	if v == "true" {
		return true
	}
	return false
}

func (p *properties) DefaultBool(key string, defaultVal string) bool {
	result := defaultVal
	if v := p.Get(key); v != "" {
		result = v
	}
	if result == "true" {
		return true
	}
	return false
}

func (part *part) initPath(path string) {
	part.vals = make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		b, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		s := strings.TrimSpace(string(b))
		if strings.Index(s, "#") == 0 {
			continue
		}
		index := strings.Index(s, "=")
		if index < 0 {
			continue
		}
		first := strings.TrimSpace(s[:index])
		if len(first) == 0 {
			continue
		}
		second := strings.TrimSpace(s[index+1:])
		pos := strings.Index(second, "\t#")
		if pos > -1 {
			second = second[0:pos]
		}
		pos = strings.Index(second, " #")
		if pos > -1 {
			second = second[0:pos]
		}
		if len(second) == 0 {
			continue
		}
		key := first
		part.vals[key] = strings.TrimSpace(second)
	}
}

func Init(path string) {
	data := new(part)
	data.initPath(path)
	if Config.values == nil {
		Config.values = make(map[string]string)
	}
	for k, v := range data.vals {
		Config.Set(k, v)
	}
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
