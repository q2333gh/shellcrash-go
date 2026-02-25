package setbootctl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Options struct {
	CrashDir string
}

type Deps struct {
	RunCommand func(name string, args ...string) error
}

type State struct {
	StartOld     string
	StartDelay   int
	NetworkCheck string
	BindDir      string
	TmpDir       string
}

func RunMenu(opts Options, deps Deps, in io.Reader, out io.Writer) error {
	opts = withDefaults(opts)
	deps = withDeps(deps)
	if in == nil || out == nil {
		return fmt.Errorf("invalid io")
	}

	reader := bufio.NewReader(in)
	for {
		st, err := LoadState(opts)
		if err != nil {
			return err
		}
		autoSet := "OFF"
		if CheckAutostart(opts, st) {
			autoSet = "ON"
		}
		mini := "OFF"
		if st.BindDir != opts.CrashDir {
			mini = "ON"
		}
		delay := "未设置"
		if st.StartDelay > 0 {
			delay = fmt.Sprintf("%d秒", st.StartDelay)
		}

		fmt.Fprintln(out, "启动设置菜单")
		fmt.Fprintf(out, "1) 开机自启动: %s\n", autoSet)
		fmt.Fprintf(out, "2) 使用保守模式: %s\n", st.StartOld)
		fmt.Fprintf(out, "3) 设置自启延时: %s\n", delay)
		fmt.Fprintf(out, "4) 启用小闪存模式: %s\n", mini)
		if st.BindDir != opts.CrashDir {
			fmt.Fprintf(out, "5) 设置小闪存目录: %s\n", st.BindDir)
		}
		fmt.Fprintf(out, "6) 自启网络检查: %s\n", st.NetworkCheck)
		fmt.Fprintln(out, "7) 查看启动日志")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应标号> ")

		choice, err := readLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if CheckAutostart(opts, st) {
				if err := DisableAutostart(opts, deps); err != nil {
					return err
				}
				fmt.Fprintln(out, "已禁止ShellCrash开机自启动！")
			} else {
				if err := EnableAutostart(opts, deps); err != nil {
					return err
				}
				fmt.Fprintln(out, "已设置ShellCrash开机自启动！")
			}
		case "2":
			if err := ToggleConservativeMode(opts, deps, st); err != nil {
				return err
			}
		case "3":
			fmt.Fprint(out, "请输入启动延迟时间（0～300秒）> ")
			raw, err := readLine(reader)
			if err != nil {
				return err
			}
			sec, err := strconv.Atoi(raw)
			if err != nil || sec < 0 || sec > 300 {
				fmt.Fprintln(out, "输入有误，或超过300秒，请重新输入！")
				continue
			}
			if err := setCfgValue(opts, "start_delay", strconv.Itoa(sec)); err != nil {
				return err
			}
			fmt.Fprintln(out, "设置成功！")
		case "4":
			if err := ToggleMiniFlash(opts, st); err != nil {
				return err
			}
		case "5":
			if st.BindDir == opts.CrashDir {
				continue
			}
			fmt.Fprintln(out, "1) 使用内存（/tmp）")
			fmt.Fprintln(out, "2) 自动选择U盘目录（/mnt下首项）")
			fmt.Fprintln(out, "3) 自定义目录")
			fmt.Fprintln(out, "0) 返回")
			fmt.Fprint(out, "请输入对应标号> ")
			sub, err := readLine(reader)
			if err != nil {
				return err
			}
			switch sub {
			case "0", "":
				continue
			case "1":
				if err := SetBindDir(opts, st.TmpDir); err != nil {
					return err
				}
			case "2":
				dir, err := firstMountDir("/mnt")
				if err != nil {
					return err
				}
				if err := SetBindDir(opts, dir); err != nil {
					return err
				}
			case "3":
				fmt.Fprint(out, "请输入目录> ")
				dir, err := readLine(reader)
				if err != nil {
					return err
				}
				if err := SetBindDir(opts, dir); err != nil {
					return err
				}
			}
		case "6":
			next := "OFF"
			if st.NetworkCheck == "OFF" {
				next = "ON"
			}
			if err := setCfgValue(opts, "network_check", next); err != nil {
				return err
			}
			fmt.Fprintf(out, "network_check=%s\n", next)
		case "7":
			data, err := os.ReadFile(filepath.Join(st.TmpDir, "ShellCrash.log"))
			if err != nil {
				fmt.Fprintln(out, "未找到相关日志！")
				continue
			}
			fmt.Fprintln(out, string(data))
			return nil
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func LoadState(opts Options) (State, error) {
	opts = withDefaults(opts)
	cfgKV, err := parseKVFile(filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg"))
	if err != nil && !os.IsNotExist(err) {
		return State{}, err
	}
	envKV, err := parseKVFile(filepath.Join(opts.CrashDir, "configs", "command.env"))
	if err != nil && !os.IsNotExist(err) {
		return State{}, err
	}

	s := State{
		StartOld:     normalizeONOFF(cfgKV["start_old"], "OFF"),
		NetworkCheck: normalizeONOFF(cfgKV["network_check"], "ON"),
		BindDir:      stripQuotes(envKV["BINDIR"]),
		TmpDir:       stripQuotes(envKV["TMPDIR"]),
	}
	if s.BindDir == "" {
		s.BindDir = opts.CrashDir
	}
	if s.TmpDir == "" {
		s.TmpDir = "/tmp/ShellCrash"
	}
	if n, err := strconv.Atoi(stripQuotes(cfgKV["start_delay"])); err == nil {
		s.StartDelay = n
	}
	return s, nil
}

func CheckAutostart(opts Options, st State) bool {
	if st.StartOld == "ON" {
		return !fileExists(filepath.Join(opts.CrashDir, ".dis_startup"))
	}
	if fileExists("/etc/rc.common") && readProcComm() == "procd" {
		if entries, err := filepath.Glob("/etc/rc.d/*shellcrash*"); err == nil && len(entries) > 0 {
			return true
		}
		return !fileExists(filepath.Join(opts.CrashDir, ".dis_startup"))
	}
	if hasCommand("systemctl") {
		out, err := exec.Command("systemctl", "is-enabled", "shellcrash.service").CombinedOutput()
		return err == nil && strings.TrimSpace(string(out)) == "enabled"
	}
	if strings.Contains(readProcComm(), "s6") {
		return fileExists("/etc/s6-overlay/s6-rc.d/user/contents.d/afstart")
	}
	if exec.Command("rc-status", "-r").Run() == nil {
		out, err := exec.Command("rc-update", "show", "default").CombinedOutput()
		return err == nil && strings.Contains(string(out), "shellcrash")
	}
	return false
}

func EnableAutostart(opts Options, deps Deps) error {
	if fileExists("/etc/rc.common") && readProcComm() == "procd" {
		_ = deps.RunCommand("/etc/init.d/shellcrash", "enable")
	}
	if hasCommand("systemctl") {
		_ = deps.RunCommand("systemctl", "enable", "shellcrash.service")
	}
	if strings.Contains(readProcComm(), "s6") {
		_ = os.MkdirAll(filepath.Dir("/etc/s6-overlay/s6-rc.d/user/contents.d/afstart"), 0o755)
		_ = os.WriteFile("/etc/s6-overlay/s6-rc.d/user/contents.d/afstart", []byte{}, 0o644)
	}
	if exec.Command("rc-status", "-r").Run() == nil {
		_ = deps.RunCommand("rc-update", "add", "shellcrash", "default")
	}
	_ = os.Remove(filepath.Join(opts.CrashDir, ".dis_startup"))
	return nil
}

func DisableAutostart(opts Options, deps Deps) error {
	if entries, err := filepath.Glob("/etc/rc.d/*shellcrash*"); err == nil {
		for _, p := range entries {
			_ = os.RemoveAll(p)
		}
	}
	if hasCommand("systemctl") {
		_ = deps.RunCommand("systemctl", "disable", "shellcrash.service")
	}
	if strings.Contains(readProcComm(), "s6") {
		_ = os.RemoveAll("/etc/s6-overlay/s6-rc.d/user/contents.d/afstart")
	}
	if exec.Command("rc-status", "-r").Run() == nil {
		_ = deps.RunCommand("rc-update", "del", "shellcrash", "default")
	}
	return os.WriteFile(filepath.Join(opts.CrashDir, ".dis_startup"), []byte{}, 0o644)
}

func ToggleConservativeMode(opts Options, deps Deps, st State) error {
	if st.StartOld == "OFF" {
		if err := DisableAutostart(opts, deps); err != nil {
			return err
		}
		if err := setCfgValue(opts, "start_old", "ON"); err != nil {
			return err
		}
		_ = deps.RunCommand(filepath.Join(opts.CrashDir, "start.sh"), "stop")
		return nil
	}
	if supportsServiceStart() {
		_ = deps.RunCommand(filepath.Join(opts.CrashDir, "start.sh"), "cronset", "ShellCrash初始化")
		if err := setCfgValue(opts, "start_old", "OFF"); err != nil {
			return err
		}
		_ = deps.RunCommand(filepath.Join(opts.CrashDir, "start.sh"), "stop")
		return nil
	}
	return fmt.Errorf("当前设备不支持以其他模式启动")
}

func ToggleMiniFlash(opts Options, st State) error {
	availKB, _ := freeKB(opts.CrashDir)
	if st.BindDir == opts.CrashDir {
		if availKB > 20480 {
			return nil
		}
		if st.StartOld != "ON" && readProcComm() == "systemd" {
			return fmt.Errorf("systemd模式下需先启用保守模式")
		}
		return SetBindDir(opts, st.TmpDir)
	}
	if availKB >= 8192 {
		_ = os.RemoveAll(st.TmpDir)
	}
	return SetBindDir(opts, opts.CrashDir)
}

func SetBindDir(opts Options, dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("empty bindir")
	}
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	envPath := filepath.Join(opts.CrashDir, "configs", "command.env")
	envKV, err := parseKVFile(envPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if envKV == nil {
		envKV = map[string]string{}
	}
	envKV["BINDIR"] = dir
	return writeKVFile(envPath, envKV)
}

func withDefaults(opts Options) Options {
	if opts.CrashDir == "" {
		opts.CrashDir = "/etc/ShellCrash"
	}
	return opts
}

func withDeps(deps Deps) Deps {
	if deps.RunCommand == nil {
		deps.RunCommand = func(name string, args ...string) error {
			return exec.Command(name, args...).Run()
		}
	}
	return deps
}

func setCfgValue(opts Options, key, value string) error {
	path := filepath.Join(opts.CrashDir, "configs", "ShellCrash.cfg")
	kv, err := parseKVFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if kv == nil {
		kv = map[string]string{}
	}
	kv[key] = value
	return writeKVFile(path, kv)
}

func normalizeONOFF(v string, fallback string) string {
	s := strings.ToUpper(strings.TrimSpace(stripQuotes(v)))
	if s == "ON" || s == "OFF" {
		return s
	}
	return fallback
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func parseKVFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	kv := map[string]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		kv[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return kv, s.Err()
}

func writeKVFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(values[k])
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func readProcComm() string {
	b, err := os.ReadFile("/proc/1/comm")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func supportsServiceStart() bool {
	comm := readProcComm()
	if strings.Contains(comm, "procd") || strings.Contains(comm, "systemd") || strings.Contains(comm, "s6") {
		return true
	}
	return exec.Command("rc-status", "-r").Run() == nil
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func freeKB(path string) (uint64, error) {
	out, err := exec.Command("df", "-k", path).Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("unexpected df output")
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 4 {
		return 0, fmt.Errorf("unexpected df fields")
	}
	return strconv.ParseUint(fields[3], 10, 64)
}

func firstMountDir(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		return filepath.Join(root, e.Name()), nil
	}
	return "", fmt.Errorf("未找到可用目录")
}
