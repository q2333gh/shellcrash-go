package tgbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"shellcrash/internal/startctl"
)

type Options struct {
	CrashDir string
}

type Deps struct {
	HTTPDo              func(*http.Request) (*http.Response, error)
	Sleep               func(time.Duration)
	RunControllerAction func(crashDir string, action string) error
}

type cfgData struct {
	Token    string
	ChatID   string
	Alias    string
	MenuPush bool
}

type tgResponse struct {
	OK     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}

type tgUpdate struct {
	UpdateID      int              `json:"update_id"`
	Message       *tgMessage       `json:"message"`
	CallbackQuery *tgCallbackQuery `json:"callback_query"`
}

type tgCallbackQuery struct {
	Data    string     `json:"data"`
	Message *tgMessage `json:"message"`
}

type tgMessage struct {
	Text     string      `json:"text"`
	Document *tgDocument `json:"document"`
	Chat struct {
		ID int64 `json:"id"`
	} `json:"chat"`
}

type tgDocument struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
}

type tgGetFileResp struct {
	OK     bool `json:"ok"`
	Result struct {
		FilePath string `json:"file_path"`
	} `json:"result"`
}

type uploadState string

const (
	uploadCore uploadState = "ts_up_core"
	uploadBak  uploadState = "ts_up_bak"
	uploadCfg  uploadState = "ts_up_ccf"
)

func Run(opts Options, deps Deps) error {
	crashDir := strings.TrimSpace(opts.CrashDir)
	if crashDir == "" {
		crashDir = "/etc/ShellCrash"
	}
	if deps.HTTPDo == nil {
		client := &http.Client{Timeout: 40 * time.Second}
		deps.HTTPDo = client.Do
	}
	if deps.Sleep == nil {
		deps.Sleep = time.Sleep
	}
	if deps.RunControllerAction == nil {
		deps.RunControllerAction = runControllerAction
	}

	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	cfg, err := loadCfg(cfgPath)
	if err != nil {
		return err
	}
	if cfg.Token == "" || cfg.ChatID == "" {
		return fmt.Errorf("TG_TOKEN/TG_CHATID is not configured")
	}
	chatID, err := strconv.ParseInt(cfg.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid TG_CHATID: %w", err)
	}

	apiRoot := "https://api.telegram.org/bot" + cfg.Token
	if cfg.MenuPush {
		_ = sendMenu(deps.HTTPDo, apiRoot, chatID, crashDir)
	}
	offset := 0
	for {
		updates, next, err := pollUpdates(deps.HTTPDo, apiRoot, offset)
		if err != nil {
			deps.Sleep(10 * time.Second)
			continue
		}
		offset = next
		for _, u := range updates {
			if err := handleUpdate(deps, apiRoot, crashDir, chatID, u); err != nil {
				// Keep bot loop alive even on one bad update.
				continue
			}
		}
	}
}

func handleUpdate(deps Deps, apiRoot, crashDir string, chatID int64, u tgUpdate) error {
	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	cfgKV, _ := parseKVFile(cfgPath)

	if u.CallbackQuery != nil && u.CallbackQuery.Message != nil {
		if u.CallbackQuery.Message.Chat.ID != chatID {
			return nil
		}
		return handleCallback(deps, apiRoot, crashDir, chatID, cfgKV, u.CallbackQuery.Data)
	}
	if u.Message == nil || u.Message.Chat.ID != chatID {
		return nil
	}
	if u.Message.Document != nil {
		return handleDocumentUpload(deps, apiRoot, crashDir, chatID, cfgKV, u.Message.Document)
	}
	text := strings.TrimSpace(u.Message.Text)
	if text == "/help" {
		return sendHelp(deps.HTTPDo, apiRoot, chatID)
	}
	alias := "crash"
	if a := strings.TrimSpace(stripQuotes(cfgKV["my_alias"])); a != "" {
		alias = a
	}
	if text == "/crash" || text == "/"+alias {
		return sendMenu(deps.HTTPDo, apiRoot, chatID, crashDir)
	}
	return nil
}

func handleCallback(deps Deps, apiRoot, crashDir string, chatID int64, cfgKV map[string]string, callback string) error {
	firewallArea := stripQuotes(cfgKV["firewall_area"])
	redirMod := stripQuotes(cfgKV["redir_mod"])
	redirModBF := stripQuotes(cfgKV["redir_mod_bf"])

	switch callback {
	case "start_redir":
		if firewallArea == "4" {
			if redirModBF == "" {
				redirModBF = "Redir"
			}
			if err := setConfigValues(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), map[string]string{
				"redir_mod": redirModBF,
			}); err != nil {
				return err
			}
			if err := deps.RunControllerAction(crashDir, "start_firewall"); err != nil {
				return err
			}
			if err := sendMessage(deps.HTTPDo, apiRoot, chatID, "已切换到"+redirModBF, nil); err != nil {
				return err
			}
		} else {
			if redirMod == "" {
				redirMod = "Redir"
			}
			if err := sendMessage(deps.HTTPDo, apiRoot, chatID, "当前已经是"+redirMod, nil); err != nil {
				return err
			}
		}
		return sendMenu(deps.HTTPDo, apiRoot, chatID, crashDir)
	case "stop_redir":
		if firewallArea != "4" {
			if redirMod == "" {
				redirMod = "Redir"
			}
			if err := setConfigValues(filepath.Join(crashDir, "configs", "ShellCrash.cfg"), map[string]string{
				"redir_mod_bf":  redirMod,
				"firewall_area": "4",
			}); err != nil {
				return err
			}
			if err := deps.RunControllerAction(crashDir, "stop_firewall"); err != nil {
				return err
			}
			if err := sendMessage(deps.HTTPDo, apiRoot, chatID, "已切换到纯净模式", nil); err != nil {
				return err
			}
		} else {
			if err := sendMessage(deps.HTTPDo, apiRoot, chatID, "当前已经是纯净模式", nil); err != nil {
				return err
			}
		}
		return sendMenu(deps.HTTPDo, apiRoot, chatID, crashDir)
	case "restart":
		if err := deps.RunControllerAction(crashDir, "restart"); err != nil {
			return err
		}
		if err := sendMessage(deps.HTTPDo, apiRoot, chatID, "服务已重启", nil); err != nil {
			return err
		}
		return sendMenu(deps.HTTPDo, apiRoot, chatID, crashDir)
	case "readlog":
		logPath := filepath.Join("/tmp/ShellCrash", "ShellCrash.log")
		lines := tailLines(logPath, 20)
		text := "日志内容如下:\n" + strings.Join(lines, "\n")
		if text == "日志内容如下:\n" {
			text = "日志内容为空"
		}
		if err := sendMessage(deps.HTTPDo, apiRoot, chatID, text, nil); err != nil {
			return err
		}
		return sendMenu(deps.HTTPDo, apiRoot, chatID, crashDir)
	case "transport":
		return sendTransportMenu(deps.HTTPDo, apiRoot, chatID, cfgKV)
	case "ts_get_log":
		if err := sendDocumentFile(deps.HTTPDo, apiRoot, chatID, filepath.Join("/tmp/ShellCrash", "ShellCrash.log"), "ShellCrash.log"); err != nil {
			_ = sendMessage(deps.HTTPDo, apiRoot, chatID, "下载日志失败: "+err.Error(), nil)
		}
		return sendMenu(deps.HTTPDo, apiRoot, chatID, crashDir)
	case "ts_get_bak":
		if err := sendConfigBackup(deps.HTTPDo, apiRoot, chatID, crashDir); err != nil {
			_ = sendMessage(deps.HTTPDo, apiRoot, chatID, "备份设置失败: "+err.Error(), nil)
		}
		return sendMenu(deps.HTTPDo, apiRoot, chatID, crashDir)
	case "ts_get_ccf":
		if err := sendCoreConfigArchive(deps.HTTPDo, apiRoot, chatID, crashDir, cfgKV); err != nil {
			_ = sendMessage(deps.HTTPDo, apiRoot, chatID, "下载配置失败: "+err.Error(), nil)
		}
		return sendMenu(deps.HTTPDo, apiRoot, chatID, crashDir)
	case "ts_up_core":
		if err := writeUploadState(crashDir, uploadCore); err != nil {
			return err
		}
		return sendMessage(deps.HTTPDo, apiRoot, chatID, "请发送需要上传的内核，支持 .tar.gz/.gz/.upx", nil)
	case "ts_up_bak":
		if err := writeUploadState(crashDir, uploadBak); err != nil {
			return err
		}
		return sendMessage(deps.HTTPDo, apiRoot, chatID, "请发送需要还原的备份文件，必须是 .tar.gz", nil)
	case "ts_up_ccf":
		if err := writeUploadState(crashDir, uploadCfg); err != nil {
			return err
		}
		ext := configExt(cfgKV)
		return sendMessage(deps.HTTPDo, apiRoot, chatID, "请发送需要上传的配置文件，必须是 ."+ext, nil)
	default:
		return nil
	}
}

func sendTransportMenu(httpDo func(*http.Request) (*http.Response, error), apiRoot string, chatID int64, cfgKV map[string]string) error {
	markup := map[string]any{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "下载日志", "callback_data": "ts_get_log"},
				{"text": "备份设置", "callback_data": "ts_get_bak"},
				{"text": "下载配置", "callback_data": "ts_get_ccf"},
			},
			{
				{"text": "上传内核", "callback_data": "ts_up_core"},
				{"text": "还原设置", "callback_data": "ts_up_bak"},
				{"text": "上传配置", "callback_data": "ts_up_ccf"},
			},
		},
	}
	ext := configExt(cfgKV)
	text := "请选择需要上传或下载的文件。\n当前配置格式: ." + ext
	return sendMessage(httpDo, apiRoot, chatID, text, markup)
}

func sendHelp(httpDo func(*http.Request) (*http.Response, error), apiRoot string, chatID int64) error {
	return sendMessage(httpDo, apiRoot, chatID, "项目地址:\nhttps://github.com/juewuy/ShellClash\n相关教程:\nhttps://juewuy.github.io", nil)
}

func sendMenu(httpDo func(*http.Request) (*http.Response, error), apiRoot string, chatID int64, crashDir string) error {
	core := "Mihomo"
	if cfgKV, err := parseKVFile(filepath.Join(crashDir, "configs", "ShellCrash.cfg")); err == nil {
		switch strings.ToLower(stripQuotes(cfgKV["crashcore"])) {
		case "singbox":
			core = "SingBox"
		case "singboxr":
			core = "SingBoxR"
		case "clash":
			core = "Clash"
		case "meta":
			core = "Mihomo"
		}
	}
	run := "未运行"
	if isRunning("CrashCore") {
		run = "正在运行"
	}
	text := fmt.Sprintf("欢迎使用ShellCrash\n%s服务%s\n请选择操作：", core, run)
	markup := map[string]any{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "启用劫持", "callback_data": "start_redir"},
				{"text": "纯净模式", "callback_data": "stop_redir"},
				{"text": "重启服务", "callback_data": "restart"},
			},
			{
				{"text": "查看日志", "callback_data": "readlog"},
				{"text": "文件传输", "callback_data": "transport"},
			},
		},
	}
	return sendMessage(httpDo, apiRoot, chatID, text, markup)
}

func pollUpdates(httpDo func(*http.Request) (*http.Response, error), apiRoot string, offset int) ([]tgUpdate, int, error) {
	u := fmt.Sprintf("%s/getUpdates?timeout=25&offset=%d", apiRoot, offset)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, offset, err
	}
	resp, err := httpDo(req)
	if err != nil {
		return nil, offset, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, offset, fmt.Errorf("telegram getUpdates status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed tgResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, offset, err
	}
	next := offset
	for _, item := range parsed.Result {
		if item.UpdateID >= next {
			next = item.UpdateID + 1
		}
	}
	return parsed.Result, next, nil
}

func sendMessage(httpDo func(*http.Request) (*http.Response, error), apiRoot string, chatID int64, text string, markup any) error {
	body := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	if markup != nil {
		body["reply_markup"] = markup
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, apiRoot+"/sendMessage", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpDo(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("telegram sendMessage status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func sendDocumentFile(httpDo func(*http.Request) (*http.Response, error), apiRoot string, chatID int64, path string, fileName string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("chat_id", strconv.FormatInt(chatID, 10)); err != nil {
		return err
	}
	part, err := w.CreateFormFile("document", fileName)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, apiRoot+"/sendDocument", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := httpDo(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("telegram sendDocument status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func sendConfigBackup(httpDo func(*http.Request) (*http.Response, error), apiRoot string, chatID int64, crashDir string) error {
	tmpDir, err := loadTmpDir(crashDir)
	if err != nil {
		return err
	}
	fileName := "configs_" + time.Now().Format("20060102_150405") + ".tar.gz"
	outPath := filepath.Join(tmpDir, fileName)
	if err := archiveDirGz(filepath.Join(crashDir, "configs"), outPath); err != nil {
		return err
	}
	defer os.Remove(outPath)
	return sendDocumentFile(httpDo, apiRoot, chatID, outPath, fileName)
}

func sendCoreConfigArchive(httpDo func(*http.Request) (*http.Response, error), apiRoot string, chatID int64, crashDir string, cfgKV map[string]string) error {
	tmpDir, err := loadTmpDir(crashDir)
	if err != nil {
		return err
	}
	ext := configExt(cfgKV)
	srcDir := filepath.Join(crashDir, ext+"s")
	fileName := ext + ".tar.gz"
	outPath := filepath.Join(tmpDir, fileName)
	if err := archiveDirGz(srcDir, outPath); err != nil {
		return err
	}
	defer os.Remove(outPath)
	return sendDocumentFile(httpDo, apiRoot, chatID, outPath, fileName)
}

func handleDocumentUpload(deps Deps, apiRoot, crashDir string, chatID int64, cfgKV map[string]string, doc *tgDocument) error {
	state, err := readUploadState(crashDir)
	if err != nil || state == "" {
		return sendMessage(deps.HTTPDo, apiRoot, chatID, "当前没有待处理的上传任务，请先点击“文件传输”。", nil)
	}
	defer clearUploadState(crashDir)

	tmpDir, err := loadTmpDir(crashDir)
	if err != nil {
		return err
	}
	downloadName := sanitizeFileName(doc.FileName)
	if downloadName == "" {
		downloadName = "upload.bin"
	}
	localPath := filepath.Join(tmpDir, downloadName)
	if err := downloadTelegramFile(deps.HTTPDo, apiRoot, localPath, doc.FileID); err != nil {
		return sendMessage(deps.HTTPDo, apiRoot, chatID, "文件下载失败: "+err.Error(), nil)
	}
	defer os.Remove(localPath)

	msg, err := processUploadedFile(deps, crashDir, cfgKV, state, localPath, downloadName)
	if err != nil {
		_ = sendMessage(deps.HTTPDo, apiRoot, chatID, msg, nil)
		return sendTransportMenu(deps.HTTPDo, apiRoot, chatID, cfgKV)
	}
	if err := sendMessage(deps.HTTPDo, apiRoot, chatID, msg, nil); err != nil {
		return err
	}
	return sendMenu(deps.HTTPDo, apiRoot, chatID, crashDir)
}

func processUploadedFile(deps Deps, crashDir string, cfgKV map[string]string, state uploadState, localPath, fileName string) (string, error) {
	switch state {
	case uploadCore:
		if !hasAnySuffix(strings.ToLower(fileName), []string{".tar.gz", ".gz", ".upx"}) {
			return "文件格式不匹配，内核上传失败。", fmt.Errorf("invalid core extension")
		}
		cfg, err := startctl.LoadConfig(crashDir)
		if err != nil {
			return "读取配置失败。", err
		}
		if err := os.MkdirAll(cfg.BinDir, 0o755); err != nil {
			return "创建内核目录失败。", err
		}
		dst := filepath.Join(cfg.BinDir, "CrashCore"+filepath.Ext(fileName))
		if strings.HasSuffix(strings.ToLower(fileName), ".tar.gz") {
			dst = filepath.Join(cfg.BinDir, "CrashCore.tar.gz")
		}
		if err := copyFile(localPath, dst, 0o755); err != nil {
			return "内核上传失败。", err
		}
		_ = os.Remove(filepath.Join(cfg.BinDir, "CrashCore"))
		_ = os.Remove(filepath.Join(cfg.TmpDir, "CrashCore"))
		if err := deps.RunControllerAction(crashDir, "restart"); err != nil {
			return "内核已上传，但重启失败。", err
		}
		return "内核更新成功，服务已重启。", nil
	case uploadBak:
		if !strings.HasSuffix(strings.ToLower(fileName), ".tar.gz") {
			return "文件格式不匹配，备份还原失败。", fmt.Errorf("invalid backup extension")
		}
		if err := extractTarGz(localPath, filepath.Join(crashDir, "configs")); err != nil {
			return "备份还原失败。", err
		}
		return "配置文件已还原，请手动重启服务。", nil
	case uploadCfg:
		ext := "." + configExt(cfgKV)
		if !strings.HasSuffix(strings.ToLower(fileName), ext) {
			return "配置文件格式不匹配，上传失败。", fmt.Errorf("invalid config extension")
		}
		dstDir := filepath.Join(crashDir, configExt(cfgKV)+"s")
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return "创建配置目录失败。", err
		}
		dst := filepath.Join(dstDir, sanitizeFileName(fileName))
		if err := copyFile(localPath, dst, 0o644); err != nil {
			return "配置上传失败。", err
		}
		return "配置文件已上传，请手动重启服务。", nil
	default:
		return "未知上传状态。", fmt.Errorf("unknown upload state")
	}
}

func downloadTelegramFile(httpDo func(*http.Request) (*http.Response, error), apiRoot, dstPath, fileID string) error {
	u := apiRoot + "/getFile?file_id=" + url.QueryEscape(fileID)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := httpDo(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("telegram getFile status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var parsed tgGetFileResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return err
	}
	if parsed.Result.FilePath == "" {
		return fmt.Errorf("empty file_path")
	}

	apiFile := strings.Replace(apiRoot, "/bot", "/file/bot", 1)
	req2, err := http.NewRequest(http.MethodGet, apiFile+"/"+strings.TrimPrefix(parsed.Result.FilePath, "/"), nil)
	if err != nil {
		return err
	}
	resp2, err := httpDo(req2)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp2.Body, 1024))
		return fmt.Errorf("telegram download status=%d body=%s", resp2.StatusCode, strings.TrimSpace(string(b)))
	}
	out, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp2.Body); err != nil {
		return err
	}
	return out.Sync()
}

func configExt(cfgKV map[string]string) string {
	if strings.Contains(strings.ToLower(stripQuotes(cfgKV["crashcore"])), "singbox") {
		return "json"
	}
	return "yaml"
}

func loadTmpDir(crashDir string) (string, error) {
	cfg, err := startctl.LoadConfig(crashDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cfg.TmpDir, 0o755); err != nil {
		return "", err
	}
	return cfg.TmpDir, nil
}

func uploadStatePath(crashDir string) (string, error) {
	tmp, err := loadTmpDir(crashDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(tmp, "tgbot_state"), nil
}

func writeUploadState(crashDir string, state uploadState) error {
	p, err := uploadStatePath(crashDir)
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(state), 0o644)
}

func readUploadState(crashDir string) (uploadState, error) {
	p, err := uploadStatePath(crashDir)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return uploadState(strings.TrimSpace(string(b))), nil
}

func clearUploadState(crashDir string) {
	p, err := uploadStatePath(crashDir)
	if err == nil {
		_ = os.Remove(p)
	}
}

func hasAnySuffix(s string, suffixes []string) bool {
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
}

func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, string(os.PathSeparator), "_")
	if name == "." || name == "" {
		return ""
	}
	return name
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}

func runControllerAction(crashDir string, action string) error {
	cfg, err := startctl.LoadConfig(crashDir)
	if err != nil {
		return err
	}
	ctl := startctl.Controller{Cfg: cfg, Runtime: startctl.DetectRuntime()}
	return ctl.Run(action, "", false)
}

func loadCfg(path string) (cfgData, error) {
	m, err := parseKVFile(path)
	if err != nil {
		return cfgData{}, err
	}
	alias := stripQuotes(m["my_alias"])
	if alias == "" {
		alias = "crash"
	}
	return cfgData{
		Token:    stripQuotes(m["TG_TOKEN"]),
		ChatID:   stripQuotes(m["TG_CHATID"]),
		Alias:    alias,
		MenuPush: strings.EqualFold(stripQuotes(m["TG_menupush"]), "ON"),
	}, nil
}

func parseKVFile(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, nil
}

func setConfigValues(path string, updates map[string]string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	used := map[string]bool{}
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, _, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		if v, exists := updates[key]; exists {
			lines[i] = key + "=" + v
			used[key] = true
		}
	}
	for key, val := range updates {
		if used[key] {
			continue
		}
		lines = append(lines, key+"="+val)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
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

func isRunning(name string) bool {
	return exec.Command("pidof", name).Run() == nil
}

func tailLines(path string, count int) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "任务") {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) <= count {
		return filtered
	}
	return filtered[len(filtered)-count:]
}
