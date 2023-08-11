package apps

import (
	"errors"
	"fmt"
	teaconst "github.com/TeaOSLab/EdgeNode/internal/const"
	executils "github.com/TeaOSLab/EdgeNode/internal/utils/exec"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
	"github.com/iwind/gosock/pkg/gosock"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// AppCmd App命令帮助
type AppCmd struct {
	product       string
	version       string
	usages        []string
	options       []*CommandHelpOption
	appendStrings []string

	directives []*Directive

	sock *gosock.Sock
}

func NewAppCmd() *AppCmd {
	return &AppCmd{
		sock: gosock.NewTmpSock(teaconst.ProcessName),
	}
}

type CommandHelpOption struct {
	Code        string
	Description string
}

// Product 产品
func (this *AppCmd) Product(product string) *AppCmd {
	this.product = product
	return this
}

// Version 版本
func (this *AppCmd) Version(version string) *AppCmd {
	this.version = version
	return this
}

// Usage 使用方法
func (this *AppCmd) Usage(usage string) *AppCmd {
	this.usages = append(this.usages, usage)
	return this
}

// Option 选项
func (this *AppCmd) Option(code string, description string) *AppCmd {
	this.options = append(this.options, &CommandHelpOption{
		Code:        code,
		Description: description,
	})
	return this
}

// Append 附加内容
func (this *AppCmd) Append(appendString string) *AppCmd {
	this.appendStrings = append(this.appendStrings, appendString)
	return this
}

// Print 打印
func (this *AppCmd) Print() {
	fmt.Println(this.product + " v" + this.version)

	fmt.Println("Usage:")
	for _, usage := range this.usages {
		fmt.Println("   " + usage)
	}

	if len(this.options) > 0 {
		fmt.Println("")
		fmt.Println("Options:")

		var spaces = 20
		var max = 40
		for _, option := range this.options {
			l := len(option.Code)
			if l < max && l > spaces {
				spaces = l + 4
			}
		}

		for _, option := range this.options {
			if len(option.Code) > max {
				fmt.Println("")
				fmt.Println("  " + option.Code)
				option.Code = ""
			}

			fmt.Printf("  %-"+strconv.Itoa(spaces)+"s%s\n", option.Code, ": "+option.Description)
		}
	}

	if len(this.appendStrings) > 0 {
		fmt.Println("")
		for _, s := range this.appendStrings {
			fmt.Println(s)
		}
	}
}

// On 添加指令
func (this *AppCmd) On(arg string, callback func()) {
	this.directives = append(this.directives, &Directive{
		Arg:      arg,
		Callback: callback,
	})
}

// Run 运行
func (this *AppCmd) Run(main func()) {
	// 获取参数
	var args = os.Args[1:]
	if len(args) > 0 {
		var mainArg = args[0]
		this.callDirective(mainArg + ":before")

		switch mainArg {
		case "-v", "version", "-version", "--version":
			this.runVersion()
			return
		case "?", "help", "-help", "h", "-h":
			this.runHelp()
			return
		case "start":
			this.runStart()
			return
		case "stop":
			this.runStop()
			return
		case "restart":
			this.runRestart()
			return
		case "status":
			this.runStatus()
			return
		}

		// 查找指令
		for _, directive := range this.directives {
			if directive.Arg == mainArg {
				directive.Callback()
				return
			}
		}

		fmt.Println("unknown command '" + mainArg + "'")

		return
	}

	// 日志
	var writer = new(LogWriter)
	writer.Init()
	logs.SetWriter(writer)

	// 运行主函数
	main()
}

// 版本号
func (this *AppCmd) runVersion() {
	fmt.Println(this.product+" v"+this.version, "(build: "+runtime.Version(), runtime.GOOS, runtime.GOARCH, teaconst.Tag+")")
}

// 帮助
func (this *AppCmd) runHelp() {
	this.Print()
}

// 启动
func (this *AppCmd) runStart() {
	var pid = this.getPID()
	if pid > 0 {
		fmt.Println(this.product+" already started, pid:", pid)
		return
	}

	_ = os.Setenv("EdgeBackground", "on")

	var cmd = exec.Command(this.exe())
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Foreground: false,
		Setsid:     true,
	}

	err := cmd.Start()
	if err != nil {
		fmt.Println(this.product+"  start failed:", err.Error())
		return
	}

	// create symbolic links
	_ = this.createSymLinks()

	fmt.Println(this.product+" started ok, pid:", cmd.Process.Pid)
}

// 停止
func (this *AppCmd) runStop() {
	var pid = this.getPID()
	if pid == 0 {
		fmt.Println(this.product + " not started yet")
		return
	}

	// 从systemd中停止
	if runtime.GOOS == "linux" {
		systemctl, _ := executils.LookPath("systemctl")
		if len(systemctl) > 0 {
			go func() {
				// 有可能会长时间执行，这里不阻塞进程
				_ = exec.Command(systemctl, "stop", teaconst.SystemdServiceName).Run()
			}()
		}
	}

	// 如果仍在运行，则发送停止指令
	_, _ = this.sock.SendTimeout(&gosock.Command{Code: "stop"}, 1*time.Second)

	fmt.Println(this.product+" stopped ok, pid:", types.String(pid))
}

// 重启
func (this *AppCmd) runRestart() {
	this.runStop()
	time.Sleep(1 * time.Second)
	this.runStart()
}

// 状态
func (this *AppCmd) runStatus() {
	var pid = this.getPID()
	if pid == 0 {
		fmt.Println(this.product + " not started yet")
		return
	}

	fmt.Println(this.product + " is running, pid: " + types.String(pid))
}

// 获取当前的PID
func (this *AppCmd) getPID() int {
	if !this.sock.IsListening() {
		return 0
	}

	reply, err := this.sock.Send(&gosock.Command{Code: "pid"})
	if err != nil {
		return 0
	}
	return maps.NewMap(reply.Params).GetInt("pid")
}

// ParseOptions 分析参数中的选项
func (this *AppCmd) ParseOptions(args []string) map[string][]string {
	var result = map[string][]string{}
	for _, arg := range args {
		var pieces = strings.SplitN(arg, "=", 2)
		var key = strings.TrimLeft(pieces[0], "- ")
		key = strings.TrimSpace(key)
		var value = ""
		if len(pieces) == 2 {
			value = strings.TrimSpace(pieces[1])
		}
		result[key] = append(result[key], value)
	}
	return result
}

func (this *AppCmd) exe() string {
	var exe, _ = os.Executable()
	if len(exe) == 0 {
		exe = os.Args[0]
	}
	return exe
}

// 创建软链接
func (this *AppCmd) createSymLinks() error {
	if runtime.GOOS != "linux" {
		return nil
	}

	var exe, _ = os.Executable()
	if len(exe) == 0 {
		return nil
	}

	var errorList = []string{}

	// bin
	{
		var target = "/usr/bin/" + teaconst.ProcessName
		old, _ := filepath.EvalSymlinks(target)
		if old != exe {
			_ = os.Remove(target)
			err := os.Symlink(exe, target)
			if err != nil {
				errorList = append(errorList, err.Error())
			}
		}
	}

	// log
	{
		var realPath = filepath.Dir(filepath.Dir(exe)) + "/logs/run.log"
		var target = "/var/log/" + teaconst.ProcessName + ".log"
		old, _ := filepath.EvalSymlinks(target)
		if old != realPath {
			_ = os.Remove(target)
			err := os.Symlink(realPath, target)
			if err != nil {
				errorList = append(errorList, err.Error())
			}
		}
	}

	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList, "\n"))
	}

	return nil
}

func (this *AppCmd) callDirective(code string) {
	for _, directive := range this.directives {
		if directive.Arg == code {
			if directive.Callback != nil {
				directive.Callback()
			}
			return
		}
	}
}
