// Package assets holds the files lore installs into a project: templates,
// workflows (agent instructions) and prompts for the decisions model.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed all:files
var files embed.FS

// FS is rooted at the files directory.
var FS = mustSub(files, "files")

func mustSub(f fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// Read returns an embedded file or panics: every path is a compile-time constant.
func Read(path string) string {
	b, err := fs.ReadFile(FS, path)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Managed lists the embedded files that are copied into lore/ and kept up to
// date by `lore upgrade`: workflows and prompts. Paths are relative to lore/.
func Managed() []string {
	var out []string
	for _, dir := range []string{"workflows", "prompts"} {
		_ = fs.WalkDir(FS, dir, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				out = append(out, p)
			}
			return nil
		})
	}
	return out
}
