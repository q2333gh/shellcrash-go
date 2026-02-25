package coreconfig

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type providerTemplateEntry struct {
	Name string
	File string
}

func RunProvidersMenu(opts MenuOptions) error {
	crashDir, tmpDir, in, out, _ := normalizeMenuOptions(opts)
	reader := bufio.NewReader(in)

	for {
		coreType, err := providerCoreType(crashDir)
		if err != nil {
			return err
		}
		entries, err := loadProviderTemplateEntries(crashDir, coreType)
		if err != nil {
			return err
		}
		currentName, err := currentProviderTemplateName(crashDir, coreType, entries)
		if err != nil {
			return err
		}

		fmt.Fprintln(out, "Providers 配置生成")
		fmt.Fprintln(out, "1) 生成包含全部提供者的配置文件")
		fmt.Fprintf(out, "2) 选择规则模版 [%s]\n", currentName)
		fmt.Fprintln(out, "3) 清理providers目录文件")
		fmt.Fprintln(out, "0) 返回")
		fmt.Fprint(out, "请输入对应数字> ")
		choice, err := readSubLine(reader)
		if err != nil {
			return err
		}
		switch choice {
		case "", "0":
			return nil
		case "1":
			if err := RunSubconverterGenerate(Options{CrashDir: crashDir, TmpDir: tmpDir}); err != nil {
				return err
			}
			fmt.Fprintln(out, "配置生成成功")
		case "2":
			if err := runProviderTemplateSelectMenu(crashDir, coreType, entries, reader, out); err != nil {
				return err
			}
		case "3":
			fmt.Fprintln(out, "将清空 providers 目录下所有内容")
			fmt.Fprint(out, "请输入(1=是/0=否)> ")
			confirm, err := readSubLine(reader)
			if err != nil {
				return err
			}
			if confirm == "1" {
				if err := os.RemoveAll(filepath.Join(crashDir, "providers")); err != nil {
					return err
				}
				fmt.Fprintln(out, "清理成功")
			}
		default:
			fmt.Fprintln(out, "输入错误")
		}
	}
}

func providerCoreType(crashDir string) (string, error) {
	cfg, err := readCfg(filepath.Join(crashDir, "configs", "ShellCrash.cfg"))
	if err != nil {
		return "", err
	}
	if stripQuotes(cfg["crashcore"]) == "singboxr" {
		return "singbox", nil
	}
	return "clash", nil
}

func runProviderTemplateSelectMenu(crashDir, coreType string, entries []providerTemplateEntry, reader *bufio.Reader, out io.Writer) error {
	fmt.Fprintln(out, "请选择在线模版:")
	for i, entry := range entries {
		fmt.Fprintf(out, "%d) %s\n", i+1, entry.Name)
	}
	fmt.Fprintln(out, "a) 使用本地模版")
	fmt.Fprintln(out, "0) 返回")
	fmt.Fprint(out, "请输入对应标号> ")
	choice, err := readSubLine(reader)
	if err != nil {
		return err
	}
	if choice == "" || choice == "0" {
		return nil
	}

	cfgPath := filepath.Join(crashDir, "configs", "ShellCrash.cfg")
	cfg, err := readCfg(cfgPath)
	if err != nil {
		return err
	}
	key := "provider_temp_" + coreType

	if choice == "a" {
		fmt.Fprint(out, "请输入模版文件绝对路径(0返回)> ")
		custom, err := readSubLine(reader)
		if err != nil {
			return err
		}
		custom = strings.TrimSpace(custom)
		if custom == "" || custom == "0" {
			return nil
		}
		info, err := os.Stat(custom)
		if err != nil || info.IsDir() {
			fmt.Fprintln(out, "输入错误，找不到对应模版文件")
			return nil
		}
		cfg[key] = custom
		if err := writeCfg(cfgPath, cfg); err != nil {
			return err
		}
		fmt.Fprintln(out, "设置成功")
		return nil
	}

	n, err := strconv.Atoi(choice)
	if err != nil || n < 1 || n > len(entries) {
		fmt.Fprintln(out, "输入错误")
		return nil
	}
	cfg[key] = entries[n-1].File
	if err := writeCfg(cfgPath, cfg); err != nil {
		return err
	}
	fmt.Fprintln(out, "设置成功")
	return nil
}

func loadProviderTemplateEntries(crashDir, coreType string) ([]providerTemplateEntry, error) {
	listPath := filepath.Join(crashDir, "configs", coreType+"_providers.list")
	if _, err := os.Stat(listPath); err != nil {
		fallback := filepath.Join(crashDir, "rules", coreType+"_providers", coreType+"_providers.list")
		if _, err2 := os.Stat(fallback); err2 == nil {
			listPath = fallback
		}
	}
	data, err := os.ReadFile(listPath)
	if err != nil {
		return nil, err
	}
	entries := make([]providerTemplateEntry, 0, 8)
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		entries = append(entries, providerTemplateEntry{
			Name: fields[0],
			File: fields[1],
		})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no provider templates found in %s", listPath)
	}
	return entries, nil
}

func currentProviderTemplateName(crashDir, coreType string, entries []providerTemplateEntry) (string, error) {
	cfg, err := readCfg(filepath.Join(crashDir, "configs", "ShellCrash.cfg"))
	if err != nil {
		return "", err
	}
	key := "provider_temp_" + coreType
	current := strings.TrimSpace(stripQuotes(cfg[key]))
	if current == "" {
		return entries[0].Name, nil
	}
	for _, entry := range entries {
		if entry.File == current {
			return entry.Name, nil
		}
	}
	return current, nil
}
