package common

import (
	"bufio"
	"fmt"
	"github.com/frkhit/logger"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ref: https://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cErr := out.Close()
		if err == nil {
			err = cErr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// ref: https://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
func FileReadLine(r *bufio.Reader) (string, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

func FileReadLines(filePath string) (lines []string, err error) {
	f, err := os.Open(filePath)
	if f != nil {
		defer f.Close()
	}
	if err != nil {
		return lines, fmt.Errorf("error opening file: %v\n", err)
	}
	r := bufio.NewReader(f)
	s, e := FileReadLine(r)
	for e == nil {
		lines = append(lines, s)
		s, e = FileReadLine(r) // todo 正常结束, 异常结束
	}
	return lines, nil
}

func ExecEcho(content string) error {
	cmd := exec.Command("/bin/sh", "-c", "echo "+content)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("fail to run echo, error is %s", err)
	}
	return nil
}

func ExecEchoClearFile(target string, ) error {
	return ExecEcho("-n > " + target)
}

func WriteStringToFile(content string, targetFile string, mode os.FileMode, isAppend bool) error {
	var f *os.File
	var err error
	if isAppend {
		f, err = os.OpenFile(targetFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, mode)
	} else {
		f, err = os.OpenFile(targetFile, os.O_CREATE|os.O_RDWR, mode)
	}
	if f != nil {
		defer f.Close()
	}
	if err != nil {
		return fmt.Errorf("fail to open file[%s], error is %s", targetFile, err)
	}
	
	if isAppend {
		f.WriteString(content)
	} else {
		f.Truncate(0)
		f.Seek(0, 0)
		f.WriteString(content)
	}
	return nil
}


func ParseHostFile(hostFile string) (map[string]string, error) {
	info := make(map[string]string)
	lines, err := FileReadLines(hostFile)
	if err != nil {
		return info, err
	}
	for _, line := range lines {
		if strings.Index(line, "#") != 0 {
			contentList := strings.Split(strings.TrimSpace(strings.Replace(line, "\t", " ", -1)), " ")
			if len(contentList) > 0 {
				ip := contentList[0]
				for _, host := range contentList[1:] {
					if len(host) > 0 {
						info[host] = ip
						break
					}
				}
			}
		}
	}
	logger.Infof("get %d record from host file\n", len(info))
	return info, nil
}
