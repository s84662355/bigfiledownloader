package bigfiledownloader

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"
)

type BigDownloader struct {
	concurrency    int
	contentLen     int64
	currentPercent func(float64)
	isStop         atomic.Bool
	stopChan       chan interface{}
	haveDownload   atomic.Uint64
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
	defer d.isStop.Store(true)
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

func (d *BigDownloader) multiDownload(strURL, filename string, contentLen int64) (err error) {
	d.setBar(contentLen)

	d.contentLen = contentLen
	partSize := contentLen / int64(d.concurrency)

	var wg sync.WaitGroup
	wg.Add(d.concurrency)
	var rangeStart int64 = 0

	errChan := make(chan error, 1)
	defer close(errChan)

	destFile, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o777)
	if err != nil {
		return errors.New(fmt.Sprint(destFile, err))
	}

	defer func() {
		if err != nil {
			os.Remove(filename)
		}
	}()

	defer destFile.Close()

	err = destFile.Truncate(contentLen)
	if err != nil {
		return err
	}

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
			err := d.downloadPartial(destFile, strURL, rangeStart+downloaded, rangeEnd, i, partSize)
			if err != nil {
				select {
				case errChan <- err:
				default:
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

func (d *BigDownloader) downloadPartial(destFile *os.File, strURL string, rangeStart int64, rangeEnd int64, i int, partSize int64) error {
	if rangeStart >= rangeEnd {
		return errors.New("rangeStart>=rangeEnd")
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
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd-1))
	client.Do(req)

	resp, err := client.Do(req)
	if err != nil {
		d.isStop.Store(true)
		return err
	}
	defer resp.Body.Close()

	seek := 0

	buf := make([]byte, partSize)
	for {
		BigConn.SetDeadline(time.Now().Add(time.Second * 18)) // 设置发送接受数据超时
		// 读取数据到缓冲区中
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			d.isStop.Store(true)
			// 处理读取错误
			return err
		}
		if n == 0 {
			return nil
			//	break
		}

		_, err = destFile.WriteAt(buf[:n], int64(i)*partSize+int64(seek))
		if err != nil {
			return errors.New(fmt.Sprint("写入文件失败", err))
		}

		seek += n

		d.haveDownload.Add(uint64(n))
		if int64(len(fileData)) >= partSize {
			return nil
		}

		if d.isStop.Load() {
			break
		}

	}

	return nil
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
