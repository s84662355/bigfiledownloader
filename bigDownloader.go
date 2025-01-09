package bigfiledownloader

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func Test() {
	err := NewBigDownloader(38, func(ddd float64) {
		fmt.Println(ddd)
	}).Download(`https://sytg-browser.oss-ap-southeast-1.aliyuncs.com/CtrlFire-version/test/updateProgram6.66.zip`, "G:\\work\\windivert-go-main\\goproxy\\dddd.zip")
   fmt.Println(err)
}

type BigDownloader struct {
	concurrency int
	contentLen int64
	currentPercent func(float64)  
	isStop         atomic.Bool
	stopChan       chan interface{}
	haveDownload atomic.Uint64
}

func NewBigDownloader(concurrency int, currentPercent func(float64)) *BigDownloader {
	d := &BigDownloader{concurrency: concurrency, currentPercent: currentPercent}
	d.isStop.Store(true)
	return d
}

func (d *BigDownloader) Download(strURL, filename string) error {
	if !d.isStop.CompareAndSwap(true, false) {
		return errors.New("正在下载")
	}
	if filename == "" {
		filename = path.Base(strURL)
	}
	resp, err := http.Head(strURL)
	if err != nil {
		return err
	}
	d.stopChan = make(chan interface{})
	defer close(d.stopChan)

	d.haveDownload.Store(0)
	if resp.StatusCode == http.StatusOK && resp.Header.Get("Accept-Ranges") == "bytes" {
		return d.multiDownload(strURL, filename, int64(resp.ContentLength))
	}

	return errors.New("请求失败或者缺少Accept-Ranges头部")
}

func (d *BigDownloader) multiDownload(strURL, filename string, contentLen int64) error {
	d.setBar(contentLen)

	d.contentLen = contentLen
	partSize := contentLen / int64(d.concurrency)
	// 创建部分文件的存放目录
	partDir := d.getPartDir(filename)
	os.Mkdir(partDir, 0o777)
	defer os.RemoveAll(partDir)
	var wg sync.WaitGroup
	wg.Add(d.concurrency)
	var rangeStart int64 = 0

	fileDatas := make([][]byte, d.concurrency)

	errChan := make(chan error, 1)
	defer close(errChan)
	for i := 0; i < d.concurrency; i++ {
		go func(i int, rangeStart int64) {
			defer recover()

			defer wg.Done()

			rangeEnd := rangeStart + partSize
			// 最后一部分，总长度不能超过 ContentLength
			if i == d.concurrency-1 {
				rangeEnd = contentLen
			}

			var downloaded int64 = 0
			fileData, err := d.downloadPartial(strURL, rangeStart+downloaded, rangeEnd, i, partSize)
			if err != nil {
				select {
				case errChan <- err:
				default:
				}
			} else {
				if len(fileData) > 0 {
					fileDatas[i] = fileData
				}
			}
		}(i, rangeStart)

		rangeStart += partSize
	}

	wg.Wait()

	if d.isStop.Load() {
		select {
		case err := <-errChan:
			return errors.New(fmt.Sprint("下载error:", err))
		default:
		}
	}

	if err := d.merge(filename, fileDatas, partSize); err != nil {
		return err
	}

	fileInfo, err := os.Stat(filename)
	if err != nil {
		return err
	}

	if fileInfo.Size() != d.contentLen {
		err = errors.New("更新包下载缺少文件")
		return err
	}

	return nil
}

func (d *BigDownloader) downloadPartial(strURL string, rangeStart int64, rangeEnd int64, i int, partSize int64) ([]byte, error) {
	if rangeStart >= rangeEnd {
		return nil, errors.New("rangeStart>=rangeEnd")
	}

	fileData := make([]byte, 0, partSize)

	var BigConn net.Conn = nil
	timeout := 20 * time.Second
	// 自定义Transport，配置连接超时和读写超时
	transport := &http.Transport{
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout, // 等待响应头的超时时间
		ExpectContinueTimeout: timeout, // 100-continue状态码的超时时间
		Dial: func(netw, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(netw, addr, timeout) // 设置建立连接超时
			if err != nil {
				return nil, err
			}

			BigConn = conn
			return conn, nil
		},
	}

	// 创建一个带有自定义Transport的http.Client
	client := &http.Client{
		// Timeout:   timeout, // 请求超时时间，包含连接、读、写
		Transport: transport,
	}

	req, err := http.NewRequest("GET", strURL, nil)
	if err != nil {
		d.isStop.Store(true)
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd-1))
	client.Do(req)

	resp, err := client.Do(req)
	if err != nil {
		d.isStop.Store(true)
		return nil, err
	}
	defer resp.Body.Close()

	buf := make([]byte,partSize)
	for {
		BigConn.SetDeadline(time.Now().Add(time.Second * 18)) // 设置发送接受数据超时
		// 读取数据到缓冲区中
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			d.isStop.Store(true)
			// 处理读取错误
			return nil, err
		}
		if n == 0 {
			return fileData, nil
			//	break
		}

		fileData = append(fileData, buf[:n]...)
		d.haveDownload.Add(uint64(n))
		if int64(len(fileData)) >= partSize {
			return fileData, nil
		}

		if d.isStop.Load() {
			break
		}

	}

	return nil, nil
}

func (d *BigDownloader) merge(filename string, fileDatas [][]byte, partSize int64) error {
	destFile, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o777)
	if err != nil {
		return errors.New(fmt.Sprint(destFile, err))
	}
	defer destFile.Close()
	size := int64(d.contentLen)  
	err = destFile.Truncate(size)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(fileDatas))

	var isRun atomic.Bool

	isRun.Store(true)

	errChan := make(chan error, 1)
	close(errChan)

	for i, fileData := range fileDatas {
		go func(i int) {

			defer wg.Done()
defer recover()
			if len(fileData) == 0 {
				if isRun.CompareAndSwap(true, false) {
					errChan <- errors.New(fmt.Sprint("数据部分", i, "数据长度为0"))
				}
				return
			}

			if isRun.Load() {
				_, err := destFile.WriteAt(fileData[:], int64(i)*partSize)
				if err != nil {
					if isRun.CompareAndSwap(true, false) {
						errChan <- errors.New(fmt.Sprint("数据部分", i, err))
					}
				}
			}
		}(i)
	}

	wg.Wait()

	select {
	case err := <-errChan:

		if err != nil {
			return err
		}
	default:
	}

	return nil
}

// getPartDir 部分文件存放的目录
func (d *BigDownloader) getPartDir(filename string) string {
	return strings.SplitN(filename, ".", 2)[0]
}

// getPartFilename 构造部分文件的名字
func (d *BigDownloader) getPartFilename(filename string, partNum int) string {
	partDir := d.getPartDir(filename)
	return fmt.Sprintf("%s/%s-%d", partDir, filename, partNum)
}

func (d *BigDownloader) setBar(length int64) {
	go func() {
		defer recover()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-d.stopChan:
				return
			case <-ticker.C:
				if d.isStop.Load() {
					return
				}
				var per float64 = float64(100 * (float64(d.haveDownload.Load()) / float64(d.contentLen)))
				d.currentPercent(per / 100)
			}
		}
	}()
}
