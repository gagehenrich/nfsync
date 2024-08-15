// nfsync
//  @ghe - 2024
//   ver - 0.03

package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	defConcurrentThreads 	= 64
	maxHashBytes          	= 1048576
)

var (
	sem    chan struct{}
	logger *log.Logger
	procFs sync.Map
)

func hash(file string) (string, error) {
    f, err := os.Open(file)
    if err != nil {
        return "", err
    }
    defer f.Close()

    h := md5.New()
    _, err = io.CopyN(h, f, maxHashBytes)
    if err != nil && err != io.EOF {
        return "", err
    }

    return hex.EncodeToString(h.Sum(nil)), nil
}

func fileSize(file string) (int64, error) {
	info, err := os.Stat(file)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func fsworker(src string, dst string, wg *sync.WaitGroup, sem chan struct{}) {
	defer wg.Done()
	sem <- struct{}{}
	defer func() { <-sem }()

	dstDir := filepath.Dir(dst)
	err := os.MkdirAll(dstDir, 0755)
	if err != nil {
		logger.Printf("| DIR | ERROR | Could not create directory %s: %v\n", dstDir, err)
		return
	}

	if _, err := os.Stat(dst); err == nil {
		srcSize, err := fileSize(src)
		if err != nil {
			logger.Printf("| FILE | ERROR | Could not get size for %s: %v\n", src, err)
			return
		}

		dstSize, err := fileSize(dst)
		if err != nil {
			logger.Printf("| FILE | ERROR | Could not get size for %s: %v\n", dst, err)
			return
		}

		if srcSize == dstSize {
			srcChksum, err := hash(src)
			if err != nil {
				logger.Printf("| FILE | ERROR | Could not calculate checksum for %s: %v\n", src, err)
				return
			}

			dstChksum, err := hash(dst)
			if err != nil {
				logger.Printf("| FILE | ERROR | Could not calculate checksum for %s: %v\n", dst, err)
				return
			}

			if srcChksum == dstChksum {
				logger.Printf("| FILE | INFO | Skipping: %s already exists and matches the source file\n", dst)
				return
			}
		}
	}

	if _, loaded := procFs.LoadOrStore(src, struct{}{}); loaded {
		logger.Printf("| INFO | Skipping: %s already processed\n", src)
		return
	}

	s, err := os.Open(src)
	if err != nil {
		logger.Printf("| FILE | ERROR | Could not open %s: %v\n", src, err)
		return
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		logger.Printf("| FILE | ERROR | Could not create %s: %v\n", dst, err)
		return
	}
	defer d.Close()

	for i := 0; i <= 1; i++ {
		_, err = io.Copy(d, s)
		if err == nil {
			break
		}
		logger.Printf("| COPY | ERROR | Attempt %d: Could not copy %s to %s: %v\n", i+1, src, dst, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		logger.Printf("| RETRY | ERROR | Could not copy %s to %s upon retry: %v\n", src, dst, err)
		return
	}

	logger.Printf("| COPY | INFO | %s copied to %s\n", src, dst)
}

func indexFs(src string, dst string, wg *sync.WaitGroup, wgFs chan struct{ src, dst string }) {
	defer wg.Done()

	err := filepath.WalkDir(src, func(srcPath string, d os.DirEntry, err error) error {
		if err != nil {
			logger.Printf("| FILE | ERROR | Could not access path %s: %v\n", srcPath, err)
			return err
		}

		if !d.IsDir() {
			relPath, err := filepath.Rel(src, srcPath)
			if err != nil {
				logger.Printf("| DIR | ERROR | Could not determine relative path: %v\n", err)
				return err
			}
			dstPath := filepath.Join(dst, relPath)
			wgFs <- struct{ src, dst string }{srcPath, dstPath}
		}
		return nil
	})
	if err != nil {
		logger.Printf("| DIR | ERROR | Error walking the source directory: %v\n", err)
	}
}

func main() {

	concurrentThreadsFlag := flag.Int("threads", defConcurrentThreads, "Number of concurrent threads")
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		fmt.Println("Usage: ./nfsync [optional: --threads <number_of_threads>] <src_dir> <dst_dir>")
		return
	}

	srcDir := args[0]
	dstDir := args[1]

	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		fmt.Println("Source directory does not exist")
		return
	}

	if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
		fmt.Println("Error creating destination directory:", err)
		return
	}

	logDir := "./log"
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		fmt.Println("Error creating log directory:", err)
		return
	}

	logFileName := fmt.Sprintf("%s/nfsync-%s.log", logDir, time.Now().Format("20060102T150405"))
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error creating log file:", err)
		return
	}
	defer logFile.Close()

	logMultiWriter := io.MultiWriter(logFile, os.Stdout)
	logger = log.New(logMultiWriter, "", log.Ldate|log.Ltime)

    logger.Printf("| PROC | INFO | Allocating number of threads: %d\n", *concurrentThreadsFlag)

	sem = make(chan struct{}, *concurrentThreadsFlag)

    logger.Printf("| FILE | INFO | Indexing files on %s\n", srcDir)

	var wg sync.WaitGroup
	wgFs := make(chan struct{ src, dst string }, *concurrentThreadsFlag)

	wg.Add(1)
	go indexFs(srcDir, dstDir, &wg, wgFs)

	go func() {
		wg.Wait()
		close(wgFs)
	}()

	for file := range wgFs {
		wg.Add(1)
		go fsworker(file.src, file.dst, &wg, sem)
	}

	wg.Wait()

	logger.Println("\n========== Transfer processing is complete ==========")
}
