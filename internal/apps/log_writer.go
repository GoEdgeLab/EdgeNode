package apps

import (
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/files"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/utils/time"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type LogWriter struct {
	fileAppender *files.Appender
}

func (this *LogWriter) Init() {
	// 创建目录
	dir := files.NewFile(Tea.LogDir())
	if !dir.Exists() {
		err := dir.Mkdir()
		if err != nil {
			log.Println("[error]" + err.Error())
		}
	}

	logFile := files.NewFile(Tea.LogFile("run.log"))

	// 打开要写入的日志文件
	appender, err := logFile.Appender()
	if err != nil {
		logs.Error(err)
	} else {
		this.fileAppender = appender
	}
}

func (this *LogWriter) Write(message string) {
	backgroundEnv, _ := os.LookupEnv("EdgeBackground")
	if backgroundEnv != "on" {
		// 文件和行号
		var file string
		var line int
		if Tea.IsTesting() {
			var callDepth = 3
			var ok bool
			_, file, line, ok = runtime.Caller(callDepth)
			if ok {
				file = this.packagePath(file)
			}
		}

		if len(file) > 0 {
			log.Println(message + " (" + file + ":" + strconv.Itoa(line) + ")")
		} else {
			log.Println(message)
		}
	}

	if this.fileAppender != nil {
		_, err := this.fileAppender.AppendString(timeutil.Format("Y/m/d H:i:s ") + message + "\n")
		if err != nil {
			log.Println("[error]" + err.Error())
		}
	}
}

func (this *LogWriter) Close() {
	if this.fileAppender != nil {
		_ = this.fileAppender.Close()
	}
}

func (this *LogWriter) packagePath(path string) string {
	var pieces = strings.Split(path, "/")
	if len(pieces) >= 2 {
		return strings.Join(pieces[len(pieces)-2:], "/")
	}
	return path
}
