package taskctl

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

	"shellcrash/internal/startctl"
)

type MenuOptions struct {
	CrashDir string
	In       io.Reader
	Out      io.Writer
	Err      io.Writer
}

type menuTask struct {
	ID      string
	Command string
	Name    string
}

type managedTask struct {
	ID   string
	Desc string
}

func RunMenu(opts MenuOptions) error {
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

	cfg, err := startctl.LoadConfig(crashDir)
	if err != nil {
		return err
	}
	if err := ensureUserTaskFile(crashDir); err != nil {
		return err
	}

	s := menuState{
		cfg:    cfg,
		in:     bufio.NewReader(in),
		out:    out,
		errOut: errOut,
		runner: Runner{CrashDir: crashDir},
	}
	return s.mainLoop()
}

type menuState struct {
	cfg    startctl.Config
	in     *bufio.Reader
	out    io.Writer
	errOut io.Writer
	runner Runner
}

func (s *menuState) mainLoop() error {
	for {
		fmt.Fprintln(s.out, "自动任务菜单")
		fmt.Fprintln(s.out, "1) 添加自动任务")
		fmt.Fprintln(s.out, "2) 管理任务列表")
		fmt.Fprintln(s.out, "3) 查看任务日志")
		fmt.Fprintln(s.out, "4) 配置日志推送")
		fmt.Fprintln(s.out, "5) 添加自定义任务")
		fmt.Fprintln(s.out, "6) 删除自定义任务")
		fmt.Fprintln(s.out, "7) 使用推荐设置")
		fmt.Fprintln(s.out, "0) 返回")
		choice := s.prompt("请输入对应标号> ")
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := s.addTaskFlow(); err != nil {
				fmt.Fprintf(s.errOut, "添加任务失败: %v\n", err)
			}
		case "2":
			if err := s.manageTaskFlow(); err != nil {
				fmt.Fprintf(s.errOut, "管理任务失败: %v\n", err)
			}
		case "3":
			s.showTaskLogs()
		case "4":
			fmt.Fprintln(s.out, "请在日志工具中配置相关推送通道及推送开关")
		case "5":
			if err := s.addCustomTaskFlow(); err != nil {
				fmt.Fprintf(s.errOut, "添加自定义任务失败: %v\n", err)
			}
		case "6":
			if err := s.deleteCustomTaskFlow(); err != nil {
				fmt.Fprintf(s.errOut, "删除自定义任务失败: %v\n", err)
			}
		case "7":
			if err := s.applyRecommendedTasks(); err != nil {
				fmt.Fprintf(s.errOut, "应用推荐设置失败: %v\n", err)
			}
		default:
			fmt.Fprintln(s.out, "输入错误")
		}
	}
}

func (s *menuState) addTaskFlow() error {
	tasks, err := loadTasks(s.cfg.CrashDir)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Fprintln(s.out, "未找到可添加任务")
		return nil
	}
	for i, t := range tasks {
		fmt.Fprintf(s.out, "%d) [%s] %s\n", i+1, t.ID, t.Name)
	}
	choice := s.prompt("请输入对应标号(0返回)> ")
	if choice == "" || choice == "0" {
		return nil
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(tasks) {
		return fmt.Errorf("invalid task selection")
	}
	return s.taskTypeFlow(tasks[idx-1].ID, tasks[idx-1].Name)
}

func (s *menuState) manageTaskFlow() error {
	items, err := collectManagedTasks(s.cfg.CrashDir)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Fprintln(s.out, "当前没有可供管理的任务")
		return nil
	}
	for i, it := range items {
		fmt.Fprintf(s.out, "%d) %s %s\n", i+1, it.ID, it.Desc)
	}
	fmt.Fprintln(s.out, "a) 清空旧版任务")
	fmt.Fprintln(s.out, "d) 清空任务列表")
	choice := s.prompt("请输入对应标号(0返回)> ")
	switch choice {
	case "", "0":
		return nil
	case "a":
		return removeTaskByKeyword(s.cfg.CrashDir, "#")
	case "d":
		return removeTaskByKeyword(s.cfg.CrashDir, "task.sh")
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(items) {
		return fmt.Errorf("invalid managed-task selection")
	}
	item := items[idx-1]
	if item.ID == "0" {
		if s.prompt("旧版任务不支持管理，是否移除？(1是/0否)> ") == "1" {
			return removeTaskByKeyword(s.cfg.CrashDir, item.Desc)
		}
		return nil
	}
	fmt.Fprintf(s.out, "当前任务: %s\n", item.Desc)
	fmt.Fprintln(s.out, "1) 修改当前任务")
	fmt.Fprintln(s.out, "2) 删除当前任务")
	fmt.Fprintln(s.out, "3) 立即执行一次")
	fmt.Fprintln(s.out, "4) 查看执行记录")
	action := s.prompt("请输入对应标号(0返回)> ")
	switch action {
	case "", "0":
		return nil
	case "1":
		name := taskNameByID(s.cfg.CrashDir, item.ID)
		if name == "" {
			name = item.Desc
		}
		if err := s.taskTypeFlow(item.ID, name); err != nil {
			return err
		}
		return removeTaskByKeyword(s.cfg.CrashDir, item.Desc)
	case "2":
		return removeTaskByKeyword(s.cfg.CrashDir, item.Desc)
	case "3":
		return s.runner.Run([]string{item.ID, item.Desc})
	case "4":
		s.showTaskLogByName(taskNameByID(s.cfg.CrashDir, item.ID))
		return nil
	default:
		return fmt.Errorf("invalid task action")
	}
}

func (s *menuState) taskTypeFlow(taskID, taskName string) error {
	fmt.Fprintf(s.out, "请选择任务【%s】执行条件:\n", taskName)
	fmt.Fprintln(s.out, "1) 每周执行")
	fmt.Fprintln(s.out, "2) 每日执行")
	fmt.Fprintln(s.out, "3) 每隔小时执行")
	fmt.Fprintln(s.out, "4) 每隔分钟执行")
	fmt.Fprintln(s.out, "5) 服务启动前执行")
	fmt.Fprintln(s.out, "6) 服务启动后执行")
	fmt.Fprintln(s.out, "7) 服务运行时执行")
	fmt.Fprintln(s.out, "8) 防火墙重启后执行")
	choice := s.prompt("请输入对应标号(0返回)> ")
	switch choice {
	case "", "0":
		return nil
	case "1":
		week := s.prompt("在每周哪天执行(0-7,逗号分隔)> ")
		hour := s.prompt("在哪个小时执行(0-23)> ")
		if week == "" || hour == "" {
			return nil
		}
		desc := "在每周" + week + "的" + hour + "点整" + taskName
		line := "0 " + hour + " * * " + week + " " + filepath.Join(s.cfg.CrashDir, "task", "task.sh") + " " + taskID + " " + desc
		return setCronTask(s.cfg.CrashDir, desc, line)
	case "2":
		hour := s.prompt("每日哪个小时执行(0-23)> ")
		minute := s.prompt("具体哪分钟执行(0-59)> ")
		if minute == "" || hour == "" {
			return nil
		}
		desc := "在每日的" + hour + "点" + minute + "分" + taskName
		line := minute + " " + hour + " * * * " + filepath.Join(s.cfg.CrashDir, "task", "task.sh") + " " + taskID + " " + desc
		return setCronTask(s.cfg.CrashDir, desc, line)
	case "3":
		hours := s.prompt("每隔多少小时执行(1-23)> ")
		if hours == "" {
			return nil
		}
		desc := "每隔" + hours + "小时" + taskName
		line := "0 */" + hours + " * * * " + filepath.Join(s.cfg.CrashDir, "task", "task.sh") + " " + taskID + " " + desc
		return setCronTask(s.cfg.CrashDir, desc, line)
	case "4":
		minutes := s.prompt("每隔多少分钟执行(1-59)> ")
		if minutes == "" {
			return nil
		}
		desc := "每隔" + minutes + "分钟" + taskName
		line := "*/" + minutes + " * * * * " + filepath.Join(s.cfg.CrashDir, "task", "task.sh") + " " + taskID + " " + desc
		return setCronTask(s.cfg.CrashDir, desc, line)
	case "5":
		return setServiceTask(s.cfg.CrashDir, "bfstart", taskID, "服务启动前"+taskName, "")
	case "6":
		return setServiceTask(s.cfg.CrashDir, "afstart", taskID, "服务启动后"+taskName, "")
	case "7":
		minutes := s.prompt("每隔多少分钟执行(1-1440)> ")
		if minutes == "" {
			return nil
		}
		n, err := strconv.Atoi(minutes)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid running interval")
		}
		timeDesc := minutes + "分钟"
		cronExpr := "*/" + minutes + " * * * *"
		if n >= 60 {
			h := n / 60
			if h < 1 {
				h = 1
			}
			timeDesc = strconv.Itoa(h) + "小时"
			cronExpr = "0 */" + strconv.Itoa(h) + " * * *"
		}
		return setServiceTask(s.cfg.CrashDir, "running", taskID, "运行时每"+timeDesc+taskName, cronExpr)
	case "8":
		if s.prompt("确认继续？(1是/0否)> ") != "1" {
			return nil
		}
		return setServiceTask(s.cfg.CrashDir, "affirewall", taskID, "防火墙重启后"+taskName, "")
	default:
		return fmt.Errorf("invalid task type")
	}
}

func (s *menuState) addCustomTaskFlow() error {
	command := strings.TrimSpace(s.prompt("请输入命令(0返回)> "))
	if command == "" || command == "0" {
		return nil
	}
	tasks, err := loadUserTasks(s.cfg.CrashDir)
	if err != nil {
		return err
	}
	maxID := 200
	for _, t := range tasks {
		if n, convErr := strconv.Atoi(t.ID); convErr == nil && n > maxID {
			maxID = n
		}
	}
	newID := strconv.Itoa(maxID + 1)
	note := strings.TrimSpace(s.prompt("请输入任务备注(留空自动生成)> "))
	if note == "" {
		note = "自定义任务" + newID
	}
	line := newID + "#" + command + "#" + note + "\n"
	path := filepath.Join(s.cfg.CrashDir, "task", "task.user")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

func (s *menuState) deleteCustomTaskFlow() error {
	tasks, err := loadUserTasks(s.cfg.CrashDir)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Fprintln(s.out, "你暂未添加任何自定义任务")
		return nil
	}
	for _, t := range tasks {
		fmt.Fprintf(s.out, "%s) %s\n", t.ID, t.Name)
	}
	id := strings.TrimSpace(s.prompt("请输入对应ID(0返回)> "))
	if id == "" || id == "0" {
		return nil
	}
	path := filepath.Join(s.cfg.CrashDir, "task", "task.user")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		if strings.HasPrefix(strings.TrimSpace(line), id+"#") {
			continue
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")+"\n"), 0o644)
}

func (s *menuState) applyRecommendedTasks() error {
	if s.prompt("是否应用推荐任务？(1是/0否)> ") != "1" {
		return nil
	}
	return ApplyRecommendedTasks(s.cfg.CrashDir)
}

func ApplyRecommendedTasks(crashDir string) error {
	crashDir = strings.TrimSpace(crashDir)
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	if err := setServiceTask(crashDir, "running", "106", "订阅保活", "*/10 * * * *"); err != nil {
		return err
	}
	if err := setServiceTask(crashDir, "afstart", "107", "重启后保存面板", ""); err != nil {
		return err
	}
	line := "0 3 * * * " + filepath.Join(crashDir, "task", "task.sh") + " 103 自动更新订阅"
	return setCronTask(crashDir, "自动更新订阅", line)
}

func (s *menuState) showTaskLogs() {
	path := filepath.Join(s.cfg.TmpDir, "ShellCrash.log")
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(s.out, "未找到任务相关执行日志")
		return
	}
	found := false
	for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if strings.Contains(line, "任务【") {
			fmt.Fprintln(s.out, line)
			found = true
		}
	}
	if !found {
		fmt.Fprintln(s.out, "未找到任务相关执行日志")
	}
}

func (s *menuState) showTaskLogByName(name string) {
	if strings.TrimSpace(name) == "" {
		fmt.Fprintln(s.out, "未找到相关执行记录")
		return
	}
	path := filepath.Join(s.cfg.TmpDir, "ShellCrash.log")
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(s.out, "未找到相关执行记录")
		return
	}
	found := false
	for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if strings.Contains(line, name) {
			fmt.Fprintln(s.out, line)
			found = true
		}
	}
	if !found {
		fmt.Fprintln(s.out, "未找到相关执行记录")
	}
}

func (s *menuState) prompt(label string) string {
	fmt.Fprint(s.out, label)
	line, _ := s.in.ReadString('\n')
	return strings.TrimSpace(line)
}

func ensureUserTaskFile(crashDir string) error {
	path := filepath.Join(crashDir, "task", "task.user")
	if fileExists(path) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	comment := "#任务ID(必须>200并顺序排列)#任务命令#任务说明(#号隔开，任务命令和说明中都不允许包含#号)\n"
	return os.WriteFile(path, []byte(comment), 0o644)
}

func loadTasks(crashDir string) ([]menuTask, error) {
	list := make([]menuTask, 0, 64)
	seen := map[string]struct{}{}
	for _, path := range []string{
		filepath.Join(crashDir, "task", "task.list"),
		filepath.Join(crashDir, "task", "task.user"),
	} {
		b, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "#", 3)
			if len(parts) != 3 {
				continue
			}
			id := strings.TrimSpace(parts[0])
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			list = append(list, menuTask{ID: id, Command: strings.TrimSpace(parts[1]), Name: strings.TrimSpace(parts[2])})
		}
	}
	sort.Slice(list, func(i, j int) bool {
		a, errA := strconv.Atoi(list[i].ID)
		b, errB := strconv.Atoi(list[j].ID)
		if errA == nil && errB == nil {
			return a < b
		}
		return list[i].ID < list[j].ID
	})
	return list, nil
}

func loadUserTasks(crashDir string) ([]menuTask, error) {
	tasks, err := loadTasks(crashDir)
	if err != nil {
		return nil, err
	}
	out := make([]menuTask, 0, len(tasks))
	for _, t := range tasks {
		idNum, convErr := strconv.Atoi(t.ID)
		if convErr == nil && idNum > 200 {
			out = append(out, t)
		}
	}
	return out, nil
}

func setCronTask(crashDir, keyword, line string) error {
	if err := runCronset(crashDir, keyword, line); err != nil {
		return err
	}
	persist := filepath.Join(crashDir, "task", "cron")
	if err := os.MkdirAll(filepath.Dir(persist), 0o755); err != nil {
		return err
	}
	old, _ := os.ReadFile(persist)
	filtered := make([]string, 0, 16)
	for _, raw := range strings.Split(strings.ReplaceAll(string(old), "\r\n", "\n"), "\n") {
		l := strings.TrimSpace(raw)
		if l == "" || strings.Contains(l, keyword) {
			continue
		}
		filtered = append(filtered, l)
	}
	filtered = append(filtered, line)
	return os.WriteFile(persist, []byte(strings.Join(filtered, "\n")+"\n"), 0o644)
}

func setServiceTask(crashDir, taskType, taskID, desc, runningCron string) error {
	file := filepath.Join(crashDir, "task", taskType)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	old, _ := os.ReadFile(file)
	keep := make([]string, 0, 16)
	for _, raw := range strings.Split(strings.ReplaceAll(string(old), "\r\n", "\n"), "\n") {
		l := strings.TrimSpace(raw)
		if l == "" || strings.Contains(l, desc) {
			continue
		}
		keep = append(keep, l)
	}
	line := filepath.Join(crashDir, "task", "task.sh") + " " + taskID + " " + desc
	if taskType == "running" {
		line = runningCron + " " + line
	}
	keep = append(keep, line)
	if err := os.WriteFile(file, []byte(strings.Join(keep, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	if taskType == "running" && isCoreRunning() {
		return runCronset(crashDir, desc, line)
	}
	return nil
}

func removeTaskByKeyword(crashDir, keyword string) error {
	if err := runCronset(crashDir, keyword, ""); err != nil {
		return err
	}
	for _, path := range []string{"cron", "bfstart", "afstart", "running", "affirewall"} {
		file := filepath.Join(crashDir, "task", path)
		b, err := os.ReadFile(file)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		lines := make([]string, 0, 16)
		for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || strings.Contains(line, keyword) {
				continue
			}
			lines = append(lines, line)
		}
		content := ""
		if len(lines) > 0 {
			content = strings.Join(lines, "\n") + "\n"
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func runCronset(crashDir, keyword, line string) error {
	args := []string{keyword}
	if strings.TrimSpace(line) != "" {
		args = append(args, line)
	}
	return startActionRunWithArgs(crashDir, "cronset", args)
}

func isCoreRunning() bool {
	return exec.Command("pidof", "CrashCore").Run() == nil
}

func collectManagedTasks(crashDir string) ([]managedTask, error) {
	out := make([]managedTask, 0, 32)
	seen := map[string]struct{}{}
	for _, path := range []string{"cron", "running", "bfstart", "afstart", "affirewall"} {
		file := filepath.Join(crashDir, "task", path)
		b, err := os.ReadFile(file)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, raw := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" {
				continue
			}
			if id, desc, ok := parseManagedTaskLine(line); ok {
				key := id + "|" + desc
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				out = append(out, managedTask{ID: id, Desc: desc})
				continue
			}
			if idx := strings.Index(line, "#"); idx >= 0 {
				desc := strings.TrimSpace(line[idx+1:])
				if desc != "" && !strings.Contains(desc, "守护") {
					key := "0|旧版任务-" + desc
					if _, exists := seen[key]; !exists {
						seen[key] = struct{}{}
						out = append(out, managedTask{ID: "0", Desc: desc})
					}
				}
			}
		}
	}
	return out, nil
}

func parseManagedTaskLine(line string) (id, desc string, ok bool) {
	marker := "task/task.sh"
	idx := strings.Index(line, marker)
	if idx < 0 {
		return "", "", false
	}
	rest := strings.TrimSpace(line[idx+len(marker):])
	if strings.HasPrefix(rest, "\"") {
		rest = strings.TrimPrefix(rest, "\"")
	}
	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return "", "", false
	}
	return fields[0], strings.Join(fields[1:], " "), true
}

func taskNameByID(crashDir, id string) string {
	if id == "" {
		return ""
	}
	_, name, err := resolveTaskCommand(crashDir, id)
	if err != nil {
		return ""
	}
	return name
}
