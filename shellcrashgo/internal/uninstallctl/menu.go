package uninstallctl

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func RunMenu(opts Options, deps Deps, in io.Reader, out io.Writer) error {
	if in == nil {
		return fmt.Errorf("nil input")
	}
	if out == nil {
		return fmt.Errorf("nil output")
	}

	reader := bufio.NewReader(in)
	fmt.Fprintln(out, "警告：该操作不可逆！")
	fmt.Fprint(out, "是否确认卸载ShellCrash？ [1/0]: ")
	confirm, err := readMenuLine(reader)
	if err != nil {
		return err
	}
	if confirm != "1" {
		fmt.Fprintln(out, "操作已取消！")
		return nil
	}

	keepConfig := false
	crashDir := strings.TrimSpace(opts.CrashDir)
	if crashDir != "" && crashDir != "/" {
		fmt.Fprint(out, "是否保留脚本配置及订阅文件？ [1/0]: ")
		keep, err := readMenuLine(reader)
		if err != nil {
			return err
		}
		keepConfig = keep == "1"
	} else {
		fmt.Fprintln(out, "环境变量配置有误，请尝试手动移除安装目录！")
	}

	runOpts := opts
	runOpts.KeepConfig = keepConfig
	if err := Run(runOpts, deps); err != nil {
		return err
	}

	fmt.Fprintln(out, "已卸载ShellCrash相关文件！有缘再会！")
	fmt.Fprintln(out, "请手动关闭当前窗口以重置环境变量！")
	return nil
}

func readMenuLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	return line, nil
}
