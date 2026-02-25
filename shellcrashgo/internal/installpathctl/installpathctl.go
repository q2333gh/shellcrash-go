package installpathctl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Options struct {
	SysType string
	In      io.Reader
	Out     io.Writer
}

type Result struct {
	Dir      string
	CrashDir string
}

func RunSelect(opts Options) (Result, error) {
	sysType, in, out := withDefaults(opts)
	r := &selector{
		sysType: sysType,
		in:      bufio.NewReader(in),
		out:     out,
	}
	return r.selectInstallDir()
}

func RunUSB(opts Options) (Result, error) {
	_, in, out := withDefaults(opts)
	r := &selector{in: bufio.NewReader(in), out: out}
	dir, err := r.chooseMount("/mnt", "请选择安装目录：")
	if err != nil {
		return Result{}, err
	}
	return buildResult(dir)
}

func RunXiaomi(opts Options) (Result, error) {
	_, in, out := withDefaults(opts)
	r := &selector{in: bufio.NewReader(in), out: out}
	dir, err := r.chooseXiaomiDir()
	if err != nil {
		return Result{}, err
	}
	return buildResult(dir)
}

func RunAsusUSB(opts Options) (Result, error) {
	_, in, out := withDefaults(opts)
	r := &selector{in: bufio.NewReader(in), out: out}
	dir, err := r.chooseMount("/tmp/mnt", "请选择U盘目录：")
	if err != nil {
		return Result{}, err
	}
	return buildResult(dir)
}

func RunAsus(opts Options) (Result, error) {
	_, in, out := withDefaults(opts)
	r := &selector{in: bufio.NewReader(in), out: out}
	dir, err := r.chooseAsusDir()
	if err != nil {
		return Result{}, err
	}
	return buildResult(dir)
}

func RunCustom(opts Options) (Result, error) {
	_, in, out := withDefaults(opts)
	r := &selector{in: bufio.NewReader(in), out: out}
	dir, err := r.chooseCustomDir()
	if err != nil {
		return Result{}, err
	}
	return buildResult(dir)
}

type selector struct {
	sysType string
	in      *bufio.Reader
	out     io.Writer
}

func (s *selector) selectInstallDir() (Result, error) {
	for {
		dir, err := s.chooseBySysType()
		if err != nil {
			return Result{}, err
		}
		fmt.Fprintf(s.out, "目标目录 %s，是否确认安装？(1是/0否)> ", dir)
		if s.prompt() == "1" {
			return buildResult(dir)
		}
	}
}

func (s *selector) chooseBySysType() (string, error) {
	switch s.sysType {
	case "Padavan":
		return "/etc/storage", nil
	case "mi_snapshot":
		return s.chooseXiaomiDir()
	case "asusrouter":
		return s.chooseAsusDir()
	case "ng_snapshot":
		return "/tmp/mnt", nil
	default:
		return s.chooseGenericDir()
	}
}

func (s *selector) chooseGenericDir() (string, error) {
	for {
		fmt.Fprintln(s.out, "1) 在 /etc 目录下安装")
		fmt.Fprintln(s.out, "2) 在 /usr/share 目录下安装")
		fmt.Fprintln(s.out, "3) 在当前用户目录下安装")
		fmt.Fprintln(s.out, "4) 在外置存储中安装")
		fmt.Fprintln(s.out, "5) 手动设置安装目录")
		fmt.Fprintln(s.out, "0) 退出安装")
		fmt.Fprint(s.out, "请输入相应数字> ")
		switch s.prompt() {
		case "1":
			return "/etc", nil
		case "2":
			return "/usr/share", nil
		case "3":
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, ".local", "share"), nil
		case "4":
			return s.chooseMount("/mnt", "请选择安装目录：")
		case "5":
			return s.chooseCustomDir()
		case "0", "":
			return "", fmt.Errorf("installation cancelled")
		default:
			fmt.Fprintln(s.out, "输入错误")
		}
	}
}

func (s *selector) chooseXiaomiDir() (string, error) {
	cands := []string{"/data", "/userdisk", "/data/other_vol"}
	list := make([]string, 0, len(cands)+1)
	for _, c := range cands {
		if isDir(c) {
			list = append(list, c)
		}
	}
	for {
		fmt.Fprintln(s.out, "检测到当前设备为小米官方系统，请选择安装位置：")
		for i, p := range list {
			fmt.Fprintf(s.out, "%d) %s\n", i+1, p)
		}
		fmt.Fprintf(s.out, "%d) 自定义目录\n", len(list)+1)
		fmt.Fprintln(s.out, "0) 退出安装")
		fmt.Fprint(s.out, "请输入相应数字> ")
		raw := s.prompt()
		if raw == "0" || raw == "" {
			return "", fmt.Errorf("installation cancelled")
		}
		idx := parseIndex(raw)
		if idx < 1 || idx > len(list)+1 {
			fmt.Fprintln(s.out, "输入错误")
			continue
		}
		if idx == len(list)+1 {
			return s.chooseCustomDir()
		}
		return list[idx-1], nil
	}
}

func (s *selector) chooseAsusDir() (string, error) {
	for {
		fmt.Fprintln(s.out, "检测到当前设备为华硕固件，请选择安装方式")
		fmt.Fprintln(s.out, "1) 基于U盘安装")
		fmt.Fprintln(s.out, "2) 基于自启脚本安装")
		fmt.Fprintln(s.out, "0) 退出安装")
		fmt.Fprint(s.out, "请输入相应数字> ")
		switch s.prompt() {
		case "1":
			return s.chooseMount("/tmp/mnt", "请选择U盘目录：")
		case "2":
			return "/jffs", nil
		case "0", "":
			return "", fmt.Errorf("installation cancelled")
		default:
			fmt.Fprintln(s.out, "输入错误")
		}
	}
}

func (s *selector) chooseMount(root, title string) (string, error) {
	for {
		entries, err := listDirs(root)
		if err != nil {
			return "", err
		}
		if len(entries) == 0 {
			return "", fmt.Errorf("no installable mount path under %s", root)
		}
		fmt.Fprintln(s.out, title)
		for i, p := range entries {
			fmt.Fprintf(s.out, "%d) %s\n", i+1, p)
		}
		fmt.Fprint(s.out, "请输入相应数字> ")
		idx := parseIndex(s.prompt())
		if idx < 1 || idx > len(entries) {
			fmt.Fprintln(s.out, "输入错误")
			continue
		}
		return entries[idx-1], nil
	}
}

func (s *selector) chooseCustomDir() (string, error) {
	for {
		fmt.Fprint(s.out, "请输入自定义路径> ")
		dir := strings.TrimSpace(s.prompt())
		if !validCustomPath(dir) {
			fmt.Fprintln(s.out, "路径错误")
			continue
		}
		if !isDir(dir) {
			fmt.Fprintln(s.out, "路径不存在")
			continue
		}
		return dir, nil
	}
}

func (s *selector) prompt() string {
	line, err := s.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return ""
	}
	return strings.TrimSpace(line)
}

func withDefaults(opts Options) (string, io.Reader, io.Writer) {
	sysType := strings.TrimSpace(opts.SysType)
	in := opts.In
	if in == nil {
		in = os.Stdin
	}
	out := opts.Out
	if out == nil {
		out = os.Stdout
	}
	return sysType, in, out
}

func buildResult(dir string) (Result, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return Result{}, fmt.Errorf("empty install dir")
	}
	return Result{
		Dir:      dir,
		CrashDir: filepath.Join(dir, "ShellCrash"),
	}, nil
}

func listDirs(root string) ([]string, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		out = append(out, filepath.Join(root, e.Name()))
	}
	sort.Strings(out)
	return out, nil
}

func parseIndex(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	n := 0
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func validCustomPath(dir string) bool {
	if !strings.HasPrefix(dir, "/") {
		return false
	}
	for _, bad := range []string{"/tmp", "/opt", "/sys"} {
		if dir == bad || strings.HasPrefix(dir, bad+"/") {
			return false
		}
	}
	return true
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
