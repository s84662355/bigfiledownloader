package bigfiledownloader

import (
	"context" 
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

 

// downReader 结构体表示一个用于下载文件的读取器，封装了下载所需的上下文、URL、范围和超时时间等信息
type downReader struct {
	// 上下文，用于控制下载过程的生命周期
	ctx context.Context
	// 要下载的文件的 URL
	strURL string
	// 下载范围的起始位置
	rangeStart int64
	// 下载范围的结束位置
	rangeEnd int64
	// 读取超时时间
	readTimeOut time.Duration
	// 响应体，用于读取下载的内容
	respBody io.ReadCloser
	// 定时器，用于设置读取超时
	timer *time.Ticker

	client  *http.Client
	resChan chan struct{}
}

// newDownReader 创建一个新的 downReader 实例，用于下载指定范围的文件内容
func newDownReader(
	// 上下文，用于控制下载过程的生命周期
	pCtx context.Context,
	// 要下载的文件的 URL
	strURL string,
	// 下载范围的起始位置
	rangeStart int64,
	// 下载范围的结束位置
	rangeEnd int64,
	// 连接和读取超时时间
	timeOut time.Duration,
) (*downReader, error) {
	// 创建 downReader 实例
	dc := &downReader{
		ctx:         pCtx,
		readTimeOut: timeOut,
		resChan:     make(chan struct{}),
	}
	// 创建一个带有超时的上下文，用于控制连接超时
	ctx, cancel := context.WithTimeout(pCtx, timeOut)
	// 确保在函数结束时取消上下文
	defer cancel()

	// 自定义 HTTP Transport，设置连接超时
	transport := &http.Transport{
		DisableKeepAlives: true,
		Dial: func(network, addr string) (net.Conn, error) {
			d := &net.Dialer{}
			return d.DialContext(ctx, network, addr)
		},
	}
	// 创建一个 HTTP 客户端，使用自定义的 Transport
	dc.client = &http.Client{
		Transport: transport,
	}
	// 创建一个新的 HTTP 请求，使用指定的上下文和 URL
	req, err := http.NewRequestWithContext(pCtx, "GET", strURL, nil)
	if err != nil {
		dc.client.CloseIdleConnections()
		return nil, fmt.Errorf("%w: %v", ErrCreateRequestFailed, err)
	}
	// 设置请求头，指定下载范围
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd-1))
	// 发送 HTTP 请求
	resp, err := dc.client.Do(req)
	if err != nil {
		dc.client.CloseIdleConnections()
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	// 将响应体赋值给 downReader 实例
	dc.respBody = resp.Body
	// 创建一个定时器，用于设置读取超时
	dc.timer = time.NewTicker(timeOut)

	return dc, nil
}

// Read 方法从响应体中读取数据，并处理超时和上下文取消的情况
func (dc *downReader) Read(p []byte) (n int, err error) {
	// 启动一个 goroutine 从响应体中读取数据
	go func() {
		// 从响应体中读取数据
		n, err = dc.respBody.Read(p)
		dc.resChan <- struct{}{}
	}()

	// 重置定时器
	dc.timer.Reset(dc.readTimeOut)

	// 等待读取结果、超时或上下文取消
	select {
	// 定时器超时，关闭读取器并返回错误
	case <-dc.timer.C:
		dc.Close()
		<-dc.resChan
		return 0, ErrReadTimeout
	// 上下文被取消，关闭读取器并返回错误
	case <-dc.ctx.Done():
		dc.Close()
		<-dc.resChan
		return 0, ErrContextTimeout
	// 收到读取结果，将数据复制到缓冲区并返回
	case <-dc.resChan:
		return
	}
}

// Close 方法关闭响应体和定时器
func (dc *downReader) Close() error {
	defer dc.client.CloseIdleConnections()
	// 停止定时器
	dc.timer.Stop()
	// 关闭响应体
	return dc.respBody.Close()
}
