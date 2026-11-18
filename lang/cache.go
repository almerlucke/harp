package lang

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/almerlucke/sndfile"
)

// The mipmaps of recordings (the lower resolutions that a sampler makes for cleaner high notes) take seconds to make and are kept in a folder: the
// first load of a file filters it, the loads after that read what was kept. The folder is harp/mipmaps in the cache folder of the user
// ($XDG_CACHE_HOME or ~/.cache), or HARP_CACHE_DIR; HARP_CACHE=off keeps nothing.

func init() { sndfile.MipMapCache = CacheDir() }

// CacheDir is the folder of the mipmaps, "" if they are not kept
func CacheDir() string {
	if os.Getenv("HARP_CACHE") == "off" {
		return ""
	}
	if d := os.Getenv("HARP_CACHE_DIR"); d != "" {
		return d
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "harp", "mipmaps")
}

// CacheSize says how many files the cache has and how many bytes they take
func CacheSize() (files int, bytes int64) {
	dir := sndfile.MipMapCache
	if dir == "" {
		return 0, 0
	}
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Ext(path) == ".mip" {
			if fi, err := d.Info(); err == nil {
				files++
				bytes += fi.Size()
			}
		}
		return nil
	})
	return
}

// ClearCache removes what is kept, and says how many files it was
func ClearCache() (int, error) {
	dir := sndfile.MipMapCache
	if dir == "" {
		return 0, nil
	}
	n, _ := CacheSize()
	if err := os.RemoveAll(dir); err != nil {
		return 0, fmt.Errorf("clearing %s: %w", dir, err)
	}
	return n, nil
}
