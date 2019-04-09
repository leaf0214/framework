package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
	cf "xianhetian.com/framework/config"
)

type level = int

const (
	ERROR = iota
	INFO
	DEBUG
)

var lvlNames = []string{
	"ERROR",
	"INFO",
	"DEBUG",
}

var writes = []io.Writer{
	LogFile,   // log文件
	os.Stdout, // os.Stderr
}

var (
	logNo      uint64                                          // log序号
	defTimeFmt = "01-02 15:04:05"                              // 默认时间
	defFmt     = "信息%[1]d %[2]s %[3]s:%[4]d ▶ %.3[5]s %[6]s"   // 默认格式
	defLevel   = cf.Config.DefaultString("log_level", "DEBUG") // 默认等级
	logPath    = cf.Config.DefaultString("log_path", "./")     // 默认路径
	LogFile, _ = os.OpenFile(logPath+"log.txt", os.O_RDWR|os.O_CREATE, 0777)
	Logger     = &logger{format: defFmt, timeFormat: defTimeFmt, minion: log.New(io.MultiWriter(writes...), "", 0)}
)

// 格式化信息结构体
type inf struct {
	id       uint64
	time     string
	filename string
	line     int
	level    string
	msg      string
}

// logger结构体：module模块名称
type logger struct {
	format     string
	timeFormat string
	minion     *log.Logger
}

func Debug(p ...interface{}) {
	Logger.debug(p...)
}

func Info(p ...interface{}) {
	Logger.info(p...)
}

func Error(p ...interface{}) {
	Logger.error(p...)
}

func (r *inf) output(format string) string {
	msg := fmt.Sprintf(format,
		r.id,       // %[1]
		r.time,     // %[2]
		r.filename, // %[3]
		r.line,     // %[4]
		r.level,    // %[5]
		r.msg,      // %[6]
	)
	if i := strings.LastIndex(msg, "%!(EXTRA"); i != -1 {
		return msg[:i]
	}
	return msg
}

func (l *logger) log(calldepth int, info *inf) error {
	return l.minion.Output(calldepth+1, info.output(l.format))
}

func (l *logger) error(p ...interface{}) {
	str := paramSel(p...)
	l.logInternal(ERROR, str, 3)
}

func (l *logger) info(p ...interface{}) {
	str := paramSel(p...)
	l.logInternal(INFO, str, 3)
}

func (l *logger) debug(p ...interface{}) {
	str := paramSel(p...)
	l.logInternal(DEBUG, str, 3)
}

func (l *logger) logInternal(lvl level, msg string, pos int) {
	if i := getLvl(defLevel); lvl > i {
		return
	}
	// Calldepth指调用的深度，为0时，打印当前调用文件及行数。为1时，打印上级调用的文件及行数，依次类推。
	_, filename, line, _ := runtime.Caller(pos)
	filename = path.Base(filename)
	info := &inf{
		id:       atomic.AddUint64(&logNo, 1),
		time:     time.Now().Format(l.timeFormat),
		level:    lvlNames[lvl],
		msg:      msg,
		filename: filename,
		line:     line,
	}
	l.log(2, info)
}

func paramSel(params ...interface{}) (format string) {
	var str []string
	if len(params) <= 0 {
		str = append(str, "nil")
		return str[0]
	}
	for _, param := range params {
		switch param.(type) {
		case error:
			e := param.(error)
			str = append(str, e.Error())
		case string:
			d := param.(string)
			if strings.Contains(d, "%v") {
				format = d
			} else {
				str = append(str, d)
			}
		case []string:
			n, _ := param.([]string)
			var param bytes.Buffer
			for _, v := range n {
				param.WriteString(fmt.Sprintf(",%v", v))
			}
			rs := []rune(param.String())
			str = append(str, string(rs[1:]))
		case map[string]string:
			m, _ := param.(map[string]string)
			var param bytes.Buffer
			for k, v := range m {
				param.WriteString(fmt.Sprintf(",%v=%v", k, v))
			}
			rs := []rune(param.String())
			str = append(str, string(rs[1:]))
		default:
			b, _ := json.Marshal(param)
			str = append(str, string(b[:]))
		}
	}
	if len(format) <= 0 {
		s := strings.Join(str, ",")
		return s
	}
	for i := 0; i < len(params)-1; i++ {
		format = strings.Replace(format, "%v", str[i], 1)
	}
	return
}

func getLvl(str string) level {
	for k, v := range lvlNames {
		if str == v {
			return k
		}
	}
	return 5
}
