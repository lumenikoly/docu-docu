// Package skills exposes the immutable skill package shipped with Toudocu.
package skills

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

const (
	SkillID       = "toudocu"
	SkillVersion  = "0.0.7"
	maxFileSize   = 2 << 20
	maxBundleSize = 10 << 20
)

//go:embed all:toudocu
var embedded embed.FS

type File struct {
	Path string
	Data []byte
	Mode fs.FileMode
}

type Bundle struct {
	ID       string
	Version  string
	Files    []File
	Checksum string
}

// Load validates and returns a fresh read-only view of the embedded package.
func Load() (Bundle, error) {
	root, err := fs.Sub(embedded, SkillID)
	if err != nil {
		return Bundle{}, err
	}
	var files []File
	total := int64(0)
	seen := map[string]bool{}
	folded := map[string]string{}
	err = fs.WalkDir(root, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		clean := path.Clean(name)
		fold := strings.ToLower(clean)
		if clean == "." || clean != name || strings.HasPrefix(clean, "../") || path.IsAbs(clean) || seen[clean] || (folded[fold] != "" && folded[fold] != clean) {
			return fmt.Errorf("invalid embedded skill path %q", name)
		}
		seen[clean] = true
		folded[fold] = clean
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > maxFileSize {
			return fmt.Errorf("invalid embedded skill file %q", name)
		}
		total += info.Size()
		if total > maxBundleSize {
			return fmt.Errorf("embedded skill exceeds size limit")
		}
		data, err := fs.ReadFile(root, name)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0o644)
		if clean == "scripts" || strings.HasPrefix(clean, "scripts/") {
			mode = 0o755
		}
		files = append(files, File{Path: clean, Data: append([]byte(nil), data...), Mode: mode})
		return nil
	})
	if err != nil {
		return Bundle{}, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	if !seen["SKILL.md"] {
		return Bundle{}, fmt.Errorf("embedded skill has no SKILL.md")
	}
	if !frontMatterName(files, SkillID) {
		return Bundle{}, fmt.Errorf("embedded SKILL.md must declare name: %s", SkillID)
	}
	h := sha256.New()
	for _, file := range files {
		_, _ = fmt.Fprintf(h, "%s\x00%d\x00", file.Path, file.Mode.Perm())
		h.Write(file.Data)
		h.Write([]byte{0})
	}
	return Bundle{ID: SkillID, Version: SkillVersion, Files: files, Checksum: hex.EncodeToString(h.Sum(nil))}, nil
}

func frontMatterName(files []File, want string) bool {
	for _, file := range files {
		if file.Path != "SKILL.md" {
			continue
		}
		lines := strings.Split(string(file.Data), "\n")
		if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
			return false
		}
		for _, line := range lines[1:] {
			if strings.TrimSpace(line) == "---" {
				break
			}
			key, value, ok := strings.Cut(line, ":")
			if ok && strings.TrimSpace(key) == "name" && strings.Trim(strings.TrimSpace(value), "\"'") == want {
				return true
			}
		}
	}
	return false
}
