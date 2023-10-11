// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rolodex

import (
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type LocalFS struct {
	baseDirectory string
	Files         map[string]*LocalFile
	parseTime     int64
	logger        *slog.Logger
	readingErrors []error
}

func (l *LocalFS) Open(name string) (fs.File, error) {
	if !filepath.IsAbs(name) {
		var absErr error
		name, absErr = filepath.Abs(filepath.Join(l.baseDirectory, name))
		if absErr != nil {
			return nil, absErr
		}
	}

	if f, ok := l.Files[name]; ok {
		return &localRolodexFile{f: f}, nil
	} else {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
}

type LocalFile struct {
	filename      string
	name          string
	extension     FileExtension
	data          string
	fullPath      string
	lastModified  time.Time
	readingErrors []error
}

func NewLocalFS(baseDir string, dirFS fs.FS) (*LocalFS, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	localFiles := make(map[string]*LocalFile)
	var allErrors []error
	walkErr := fs.WalkDir(dirFS, ".", func(p string, d fs.DirEntry, err error) error {

		// we don't care about directories.
		if d.IsDir() {
			return nil
		}

		extension := ExtractFileType(p)
		var readingErrors []error
		abs, absErr := filepath.Abs(filepath.Join(baseDir, p))
		if absErr != nil {
			readingErrors = append(readingErrors, absErr)
			logger.Error("cannot create absolute path for file: ", "file", p, "error", absErr.Error())
		}

		var fileData []byte

		switch extension {
		case YAML, JSON:

			file, readErr := dirFS.Open(p)
			modTime := time.Now()
			if readErr != nil {
				readingErrors = append(readingErrors, readErr)
				allErrors = append(allErrors, readErr)
				logger.Error("cannot open file: ", "file", abs, "error", readErr.Error())
				return nil
			}
			stat, statErr := file.Stat()
			if statErr != nil {
				readingErrors = append(readingErrors, statErr)
				allErrors = append(allErrors, statErr)
				logger.Error("cannot stat file: ", "file", abs, "error", statErr.Error())
			}
			if stat != nil {
				modTime = stat.ModTime()
			}
			fileData, readErr = io.ReadAll(file)
			if readErr != nil {
				readingErrors = append(readingErrors, readErr)
				allErrors = append(allErrors, readErr)
				logger.Error("cannot read file data: ", "file", abs, "error", readErr.Error())
				return nil
			}

			logger.Debug("collecting JSON/YAML file", "file", abs)
			localFiles[abs] = &LocalFile{
				filename:      p,
				name:          filepath.Base(p),
				extension:     ExtractFileType(p),
				data:          string(fileData),
				fullPath:      abs,
				lastModified:  modTime,
				readingErrors: readingErrors,
			}
		case UNSUPPORTED:
			logger.Debug("skipping non JSON/YAML file", "file", abs)
		}
		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	return &LocalFS{
		Files:         localFiles,
		logger:        logger,
		baseDirectory: baseDir,
		readingErrors: allErrors,
	}, nil
}

func (l *LocalFile) FullPath() string {
	return l.fullPath
}

func (l *LocalFile) Name() string {
	return l.name
}

func (l *LocalFile) Size() int64 {
	return int64(len(l.data))
}

func (l *LocalFile) Mode() fs.FileMode {
	return fs.FileMode(0)
}

func (l *LocalFile) ModTime() time.Time {
	return l.lastModified
}

func (l *LocalFile) IsDir() bool {
	return false
}

func (l *LocalFile) Sys() interface{} {
	return nil
}

type localRolodexFile struct {
	f      *LocalFile
	offset int64
}

func (r *localRolodexFile) Close() error               { return nil }
func (r *localRolodexFile) Stat() (fs.FileInfo, error) { return r.f, nil }
func (r *localRolodexFile) Read(b []byte) (int, error) {
	if r.offset >= int64(len(r.f.data)) {
		return 0, io.EOF
	}
	if r.offset < 0 {
		return 0, &fs.PathError{Op: "read", Path: r.f.name, Err: fs.ErrInvalid}
	}
	n := copy(b, r.f.data[r.offset:])
	r.offset += int64(n)
	return n, nil
}
