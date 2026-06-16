package services

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// copyDir recursively copies the directory tree at src into dst, creating dst
// (and parents) as needed. Replaces the Rust copy_dir / fs_extra helper.
func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

// relaunchSelf spawns a detached copy of the current executable. The caller is
// expected to quit immediately afterwards (replaces plugin-process relaunch).
func relaunchSelf() {
	exe, err := os.Executable()
	if err != nil {
		return
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Start()
}

// systemLocale returns a best-effort BCP-47 locale (e.g. "zh-CN"). It reads the
// standard POSIX env vars; Windows falls back to "en-US" (the user can change
// language in Preferences, which is persisted). Replaces tauri-plugin-locale.
func systemLocale() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"} {
		value := os.Getenv(key)
		if value == "" || value == "C" || value == "POSIX" {
			continue
		}

		// e.g. "zh_CN.UTF-8" -> "zh-CN"
		value = strings.SplitN(value, ".", 2)[0]
		value = strings.SplitN(value, ":", 2)[0]
		value = strings.ReplaceAll(value, "_", "-")

		if value != "" {
			return value
		}
	}

	return "en-US"
}
