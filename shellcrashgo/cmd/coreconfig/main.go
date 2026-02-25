package main

import (
	"flag"
	"fmt"
	"os"

	"shellcrash/internal/coreconfig"
)

func main() {
	crashDir := flag.String("crashdir", os.Getenv("CRASHDIR"), "ShellCrash root directory")
	tmpDir := flag.String("tmpdir", os.Getenv("TMPDIR"), "ShellCrash tmp directory")
	flag.Parse()

	action := "run"
	if len(flag.Args()) > 0 {
		action = flag.Args()[0]
	}

	switch action {
	case "menu":
		if err := coreconfig.RunMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "subconverter":
		if err := coreconfig.RunSubconverterMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "subconverter-generate":
		if err := coreconfig.RunSubconverterGenerate(coreconfig.Options{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "subconverter-exclude":
		if err := coreconfig.RunSubconverterExcludeMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "subconverter-include":
		if err := coreconfig.RunSubconverterIncludeMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "subconverter-rule":
		if err := coreconfig.RunSubconverterRuleMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "subconverter-server":
		if err := coreconfig.RunSubconverterServerMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "subconverter-ua":
		if err := coreconfig.RunSubconverterUAMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "providers":
		if err := coreconfig.RunProvidersMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "providers-generate-clash":
		if err := coreconfig.RunProvidersGenerateClash(coreconfig.Options{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
		}, flag.Args()[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "providers-generate-singbox":
		if err := coreconfig.RunProvidersGenerateSingbox(coreconfig.Options{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
		}, flag.Args()[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "override-rules":
		if err := coreconfig.RunOverrideRulesMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "override":
		if err := coreconfig.RunOverrideMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "override-groups":
		if err := coreconfig.RunOverrideGroupsMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "override-proxies":
		if err := coreconfig.RunOverrideProxiesMenu(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "override-clash-adv":
		if err := coreconfig.RunOverrideClashAdvanced(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "override-singbox-adv":
		if err := coreconfig.RunOverrideSingboxAdvanced(coreconfig.MenuOptions{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
			In:       os.Stdin,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		res, err := coreconfig.Run(coreconfig.Options{
			CrashDir: *crashDir,
			TmpDir:   *tmpDir,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("ok: generated %s\n", res.CoreConfig)
	}
}
