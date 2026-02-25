package menuctl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"shellcrash/internal/coreconfig"
	"shellcrash/internal/gatewayctl"
	"shellcrash/internal/setbootctl"
	"shellcrash/internal/settingsctl"
	"shellcrash/internal/startctl"
	"shellcrash/internal/taskctl"
	"shellcrash/internal/toolsctl"
	"shellcrash/internal/uninstallctl"
	"shellcrash/internal/upgradectl"
)

type Options struct {
	CrashDir string
	In       io.Reader
	Out      io.Writer
	Err      io.Writer
}

var menuRunStartAction = RunStartAction
var menuRunSettings = func(opts settingsctl.Options, in io.Reader, out io.Writer) error {
	return settingsctl.RunMenu(opts, in, out)
}
var menuRunSetboot = func(opts setbootctl.Options, in io.Reader, out io.Writer) error {
	return setbootctl.RunMenu(opts, setbootctl.Deps{}, in, out)
}
var menuRunTask = func(opts taskctl.MenuOptions) error {
	return taskctl.RunMenu(opts)
}
var menuRunCoreConfig = func(opts coreconfig.MenuOptions) error {
	return coreconfig.RunMenu(opts)
}
var menuRunGateway = func(opts gatewayctl.Options, in io.Reader, out io.Writer) error {
	return gatewayctl.RunGatewayMenu(opts, in, out)
}
var menuRunTools = func(opts toolsctl.Options, in io.Reader, out io.Writer) error {
	return toolsctl.RunToolsMenu(opts, in, out)
}
var menuRunUpgrade = func(opts upgradectl.Options, in io.Reader, out io.Writer) error {
	return upgradectl.RunUpgradeMenu(opts, in, out)
}
var menuRunUninstall = func(opts uninstallctl.Options, in io.Reader, out io.Writer) error {
	return uninstallctl.RunMenu(opts, uninstallctl.Deps{}, in, out)
}

func Run(opts Options) error {
	crashDir := strings.TrimSpace(opts.CrashDir)
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	in := opts.In
	if in == nil {
		in = os.Stdin
	}
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.Err
	if errOut == nil {
		errOut = os.Stderr
	}

	reader := bufio.NewReader(in)
	for {
		fmt.Fprintln(out, "ShellCrash 主菜单")
		fmt.Fprintln(out, "1) 启动服务")
		fmt.Fprintln(out, "2) 基础设置")
		fmt.Fprintln(out, "3) 停止服务")
		fmt.Fprintln(out, "4) 启动设置")
		fmt.Fprintln(out, "5) 自动任务")
		fmt.Fprintln(out, "6) 配置文件")
		fmt.Fprintln(out, "7) 访问控制")
		fmt.Fprintln(out, "8) 工具优化")
		fmt.Fprintln(out, "9) 更新维护")
		fmt.Fprintln(out, "0) 退出")
		fmt.Fprint(out, "请输入对应标号> ")

		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := menuRunStartAction(crashDir, "start"); err != nil {
				fmt.Fprintf(errOut, "启动失败: %v\n", err)
			}
		case "2":
			if err := menuRunSettings(settingsctl.Options{CrashDir: crashDir}, reader, out); err != nil {
				fmt.Fprintf(errOut, "设置菜单失败: %v\n", err)
			}
		case "3":
			if err := menuRunStartAction(crashDir, "stop"); err != nil {
				fmt.Fprintf(errOut, "停止失败: %v\n", err)
			}
		case "4":
			if err := menuRunSetboot(setbootctl.Options{CrashDir: crashDir}, reader, out); err != nil {
				fmt.Fprintf(errOut, "启动设置失败: %v\n", err)
			}
		case "5":
			if err := menuRunTask(taskctl.MenuOptions{CrashDir: crashDir, In: reader, Out: out, Err: errOut}); err != nil {
				fmt.Fprintf(errOut, "任务菜单失败: %v\n", err)
			}
		case "6":
			tmpDir := "/tmp/ShellCrash"
			if cfg, err := startctl.LoadConfig(crashDir); err == nil && strings.TrimSpace(cfg.TmpDir) != "" {
				tmpDir = cfg.TmpDir
			}
			if err := menuRunCoreConfig(coreconfig.MenuOptions{
				CrashDir: crashDir,
				TmpDir:   tmpDir,
				In:       reader,
				Out:      out,
				Err:      errOut,
			}); err != nil {
				fmt.Fprintf(errOut, "配置菜单失败: %v\n", err)
			}
		case "7":
			if err := menuRunGateway(gatewayctl.Options{CrashDir: crashDir}, reader, out); err != nil {
				fmt.Fprintf(errOut, "访问控制菜单失败: %v\n", err)
			}
		case "8":
			if err := menuRunTools(toolsctl.Options{CrashDir: crashDir}, reader, out); err != nil {
				fmt.Fprintf(errOut, "工具菜单失败: %v\n", err)
			}
		case "9":
			if err := menuRunUpgrade(upgradectl.Options{CrashDir: crashDir}, reader, out); err != nil {
				fmt.Fprintf(errOut, "更新菜单失败: %v\n", err)
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func RunStartAction(crashDir, action string, extraArgs ...string) error {
	cfg, err := startctl.LoadConfig(crashDir)
	if err != nil {
		return err
	}
	ctl := startctl.Controller{Cfg: cfg, Runtime: startctl.DetectRuntime()}
	if action == "debug" {
		level := ""
		flash := false
		if len(extraArgs) > 0 {
			level = strings.TrimSpace(extraArgs[0])
		}
		if len(extraArgs) > 1 {
			flash = strings.TrimSpace(extraArgs[1]) == "flash"
		}
		return ctl.RunWithArgs(action, level, flash, nil)
	}
	return ctl.RunWithArgs(action, "", false, extraArgs)
}

func RunUninstallMenu(crashDir, binDir, alias string, in io.Reader, out io.Writer) error {
	return menuRunUninstall(uninstallctl.Options{
		CrashDir: crashDir,
		BinDir:   binDir,
		Alias:    alias,
	}, in, out)
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimSpace(strings.TrimRight(line, "\r\n")), nil
}
