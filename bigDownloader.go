package bigfiledownloader

import (
	// 用于创建和管理上下文，控制请求的生命周期
	"context"
	// 用于格式化输出信息
	"fmt"
	// 用于处理输入输出操作
	"io"
	// 用于进行数学运算
	"math"
	// 用于发起 HTTP 请求
	"net/http"
	// 用于操作文件系统
	"os"
	// 用于处理文件路径
	"path"
	// 用于实现原子操作，保证并发安全
	"sync/atomic"
	// 用于处理时间相关操作，如计时、延迟等
	"time"

	// 用于实现错误组，管理多个 goroutine 的错误
	"golang.org/x/sync/errgroup"
)

// BigDownloader 结构体表示一个大文件下载器，包含并发数、文件总长度、下载进度回调函数等信息
type BigDownloader struct {
	// 并发下载的数量
	concurrency int
	// 文件的总长度
	contentLen int64
	// 下载进度回调函数，接收一个浮点数表示下载百分比
	currentPercent func(float64)
	// 已经下载的字节数，使用原子操作保证并发安全
	haveDownload atomic.Uint64
	// 标记下载是否正在进行，使用原子操作保证并发安全
	isSart atomic.Bool
}

// NewBigDownloader 创建一个新的 BigDownloader 实例
func NewBigDownloader(
	// 并发下载的数量
	concurrency int,
	// 下载进度回调函数
	currentPercent func(float64),
) *BigDownloader {
	// 创建 BigDownloader 实例
	d := &BigDownloader{
		concurrency:    concurrency,
		currentPercent: currentPercent,
	}
	// 标记下载未开始
	d.isSart.Store(false)

	return d
}

// Download 方法用于下载大文件，支持并发下载
func (d *BigDownloader) Download(
	// 上下文，用于控制下载过程的生命周期
	ctx context.Context,
	// 要下载的文件的 URL
	strURL string,
	// 保存文件的文件名
	filename string,
) error {
	// 检查下载是否已经在进行中，如果是则返回错误
	if !d.isSart.CompareAndSwap(false, true) {
		return fmt.Errorf("正在下载")
	}
	// 确保在下载完成后标记下载未开始
	defer d.isSart.Store(false)

	// 如果文件名未指定，则使用 URL 的最后一部分作为文件名
	if filename == "" {
		filename = path.Base(strURL)
	}
	// 发送 HTTP HEAD 请求，获取文件信息
	resp, err := http.Head(strURL)
	if err != nil {
		return err
	}
	// 确保在函数结束时关闭响应体
	defer resp.Body.Close()
	// 重置已下载的字节数
	d.haveDownload.Store(0)
	// 检查响应状态码是否为 200 且服务器支持范围请求
	if resp.StatusCode == http.StatusOK && resp.Header.Get("Accept-Ranges") == "bytes" {
		// 关闭响应体
		resp.Body.Close()
		// 调用 multiDownload 方法进行并发下载
		err := d.multiDownload(
			ctx,
			strURL,
			filename,
			int64(resp.ContentLength),
		)

		fmt.Printf("下载结束------%+v \n", err)
		return err
	}

	return fmt.Errorf("请求失败或者缺少Accept-Ranges头部")
}

// multiDownload 方法用于并发下载大文件，将文件分成多个部分进行下载
func (d *BigDownloader) multiDownload(
	// 上下文，用于控制下载过程的生命周期
	pCtx context.Context,
	// 要下载的文件的 URL
	strURL string,
	// 保存文件的文件名
	filename string,
	// 文件的总长度
	contentLen int64,
) (err error) {
	// 记录文件的总长度
	d.contentLen = contentLen

	// 打开或创建目标文件，用于保存下载的内容
	destFile, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("OpenFile 文件%s失败 err = %w", filename, err)
	}

	// 下载失败就删除文件
	defer func() {
		// 关闭文件
		destFile.Close()
		if err != nil {
			// 删除文件
			os.Remove(filename)
		}
	}()

	// 调整文件大小为文件的总长度
	if err := destFile.Truncate(contentLen); err != nil {
		return fmt.Errorf("destFile Truncate err:%w", err)
	}

	// 计算每个部分的大小
	var (
		partSize         = contentLen / int64(d.concurrency)
		rangeStart int64 = 0
		// 创建一个带有错误组的上下文，用于管理多个 goroutine 的错误
		wg, ctx = errgroup.WithContext(pCtx)
	)

	// 启动进度条更新协程
	done := d.setBar()
	// 确保在函数结束时等待进度条更新协程退出
	defer func() {
		for range done {
		}
	}()

	// 启动多个 goroutine 并发下载文件的不同部分
	for i := 0; i < d.concurrency; i++ {
		ii := i
		rangeStartt := rangeStart
		wg.Go(func() error {
			return d.downloadPartial(
				ctx,
				destFile,
				ii,
				rangeStartt,
				partSize,
				strURL,
			)
		})
		// 更新下一个部分的起始位置
		rangeStart += partSize
	}

	// 等待所有 goroutine 完成，并检查是否有错误
	if err := wg.Wait(); err != nil {
		return fmt.Errorf("下载 errChan err:%w", err)
	}

	return nil
}

// downloadPartial 方法用于下载文件的一个部分
func (d *BigDownloader) downloadPartial(
	// 上下文，用于控制下载过程的生命周期
	ctx context.Context,
	// 目标文件，用于保存下载的内容
	destFile *os.File,
	// 当前部分的编号
	i int,
	// 当前部分的起始位置
	rangeStart,
	// 当前部分的大小
	partSize int64,
	// 要下载的文件的 URL
	strURL string,
) error {
	// 计算当前部分的结束位置
	rangeEnd := rangeStart + partSize
	// 如果是最后一个部分，则结束位置为文件的总长度
	if i == d.concurrency-1 {
		rangeEnd = d.contentLen
	}
	// 创建一个新的下载连接
	conn, err := d.newDownConn(
		ctx,
		strURL,
		rangeStart,
		rangeEnd,
		60*time.Second,
	)
	if err != nil {
		return err
	}
	// 确保在函数结束时关闭连接
	defer conn.Close()
	// 调用 downloadcopy 方法将下载的内容复制到目标文件中
	return d.downloadcopy(
		destFile,
		conn,
		int64(i)*partSize,
	)
}

// newDownConn 方法用于创建一个新的下载连接
func (d *BigDownloader) newDownConn(
	// 上下文，用于控制下载过程的生命周期
	ctx context.Context,
	// 要下载的文件的 URL
	strURL string,
	// 下载范围的起始位置
	rangeStart int64,
	// 下载范围的结束位置
	rangeEnd int64,
	// 读取超时时间
	readTimeOut time.Duration,
) (io.ReadCloser, error) {
	// 检查下载范围是否合法
	if rangeStart >= rangeEnd {
		return nil, fmt.Errorf("rangeStart>=rangeEnd")
	}

	// 创建一个新的下载读取器
	conn, err := newDownReader(
		ctx,
		strURL,
		rangeStart,
		rangeEnd,
		readTimeOut,
	)
	if err != nil {
		return nil, fmt.Errorf("create new conn err = %w", err)
	}
	return conn, err
}

// downloadcopy 方法用于将下载的内容复制到目标文件中
func (d *BigDownloader) downloadcopy(
	// 目标文件，用于保存下载的内容
	destFile *os.File,
	// 下载连接，用于读取下载的内容
	conn io.ReadCloser,
	// 当前部分在目标文件中的起始位置
	partSize int64,
) error {
	// 创建一个新的下载写入器
	writer := newDownWriter(
		destFile,
		partSize,
		func(n int64) {
			// 更新已下载的字节数
			d.haveDownload.Add(uint64(n))
		})
	// 将下载的内容从连接复制到写入器中
	_, err := io.Copy(writer, conn)
	return err
}

// setBar 方法用于启动一个协程，定期更新下载进度
func (d *BigDownloader) setBar() <-chan struct{} {
	// 创建一个用于通知协程退出的通道
	done := make(chan struct{})

	go func() {
		// 确保在协程结束时关闭通道
		defer close(done)
		// 创建一个定时器，每 500 毫秒触发一次
		ticker := time.NewTicker(500 * time.Millisecond)
		// 确保在协程结束时停止定时器
		defer ticker.Stop()
		for {
			select {
			// 收到退出通知，退出协程
			case done <- struct{}{}:
				return
			// 定时器触发，更新下载进度
			case <-ticker.C:
				if d.haveDownload.Load() > 0 {
					// 计算下载百分比
					var per float64 = 100 * float64((float64(d.haveDownload.Load()) / float64(d.contentLen)))
					// 四舍五入保留两位小数
					per = math.Round(per*100) / 100
					// 调用回调函数更新下载进度
					d.currentPercent(per)
				}
			}
		}
	}()

	return done
}
