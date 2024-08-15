package files

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"git.burning.moe/celediel/gt/internal/dirs"
	"github.com/charmbracelet/log"
)

const (
	directorysizes     = "directorysizes"
	length         int = 3
)

var (
	loadedDirSizes directorySizes
)

func init() {
	loadedDirSizes = readDirectorySizesFromFile()
}

type directorySize struct {
	size  int64
	mtime int64
	name  string
}

type directorySizes map[string]directorySize

func WriteDirectorySizes() {
	loadedDirSizes = updateDirectorySizes(loadedDirSizes)
	writeDirectorySizes(loadedDirSizes)
}

func readDirectorySizesFromFile() directorySizes {
	dirSizes := directorySizes{}
	for _, trash := range getAllTrashes() {
		dsf := filepath.Join(trash, directorysizes)
		if _, err := os.Lstat(dsf); os.IsNotExist(err) {
			continue
		}

		file, err := os.Open(dsf)
		if err != nil {
			log.Error(err)
			continue
		}
		defer file.Close()

		var (
			size  int64
			mtime int64
			name  string
		)

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			split := strings.Split(line, " ")
			if len(split) != length {
				log.Errorf("malformed line '%s' in %s", line, dsf)
				continue
			}

			size, err = strconv.ParseInt(split[0], 10, 64)
			if err != nil {
				log.Errorf("size %s can't be int?", split[0])
				continue
			}

			mtime, err = strconv.ParseInt(split[1], 10, 64)
			if err != nil {
				log.Errorf("mtime %s can't be int?", split[1])
				continue
			}

			name = dirs.PercentDecode(split[2])

			dirSize := directorySize{
				size:  size,
				mtime: mtime,
				name:  name,
			}
			dirSizes[name] = dirSize
		}
	}

	return dirSizes
}

func updateDirectorySizes(ds directorySizes) directorySizes {
	newDs := directorySizes{}
	for k, v := range ds {
		newDs[k] = v
	}
	for _, trash := range getAllTrashes() {
		files, err := os.ReadDir(filepath.Join(trash, "files"))
		if err != nil {
			log.Error(err)
			continue
		}
		for _, file := range files {
			if _, ok := loadedDirSizes[file.Name()]; ok {
				continue
			}

			info, err := file.Info()
			if err != nil {
				log.Error(err)
				continue
			}

			if !info.IsDir() {
				continue
			}

			newDs[info.Name()] = directorySize{
				size:  calculateDirSize(filepath.Join(trash, "files", info.Name())),
				mtime: info.ModTime().Unix(),
				name:  info.Name(),
			}
		}
	}
	return newDs
}

func writeDirectorySizes(dirSizes directorySizes) {
	// TODO: make this less bad
	for _, trash := range getAllTrashes() {
		var lines []string
		out := filepath.Join(trash, directorysizes)
		files, err := os.ReadDir(filepath.Join(trash, "files"))
		if err != nil {
			log.Error(err)
			continue
		}
		for _, file := range files {
			if dirSize, ok := dirSizes[file.Name()]; ok {
				lines = append(lines, fmt.Sprintf("%d %d ", dirSize.size, dirSize.mtime)+dirs.PercentEncode(file.Name()))
			}
		}

		err = os.WriteFile(out, []byte(strings.Join(lines, "\n")), noExecutePerm)
		if err != nil {
			log.Error(err)
		}
	}
}
