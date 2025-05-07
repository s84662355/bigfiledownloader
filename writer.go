package bigfiledownloader

import (
	// 用于处理输入输出操作
	"io"
	// 用于实现同步机制，如互斥锁
	"sync"
)

// downWriter 结构体表示一个用于下载文件的写入器，它封装了一个 io.WriterAt 接口和偏移量等信息
type downWriter struct {
	// 实现了 io.WriterAt 接口的对象，用于将数据写入到指定位置
	f io.WriterAt
	// 当前写入的偏移量，即数据将被写入的位置
	off int64
	// 互斥锁，用于保证并发写入时的线程安全
	mu sync.Mutex
	// 回调函数，在每次写入数据后调用，参数为写入的字节数
	fc func(n int64)
}

// newDownWriter 创建一个新的 downWriter 实例
func newDownWriter(
	// 实现了 io.WriterAt 接口的对象，用于将数据写入到指定位置
	f io.WriterAt,
	// 当前写入的偏移量，即数据将被写入的位置
	off int64,
	// 回调函数，在每次写入数据后调用，参数为写入的字节数
	fc func(n int64),
) *downWriter {
	// 创建 downWriter 实例
	w := &downWriter{
		f:   f,
		off: off,
		fc:  fc,
	}
	return w
}

// Write 方法将数据写入到指定的偏移位置，并更新偏移量，同时调用回调函数
func (w *downWriter) Write(p []byte) (n int, err error) {
	// 加锁，保证并发写入时的线程安全
	w.mu.Lock()
	// 调用 io.WriterAt 的 WriteAt 方法，将数据写入到指定的偏移位置
	n, err = w.f.WriteAt(p, w.off)
	// 更新偏移量
	w.off += int64(n)
	// 解锁
	w.mu.Unlock()
	// 如果回调函数不为空，则调用回调函数，传入写入的字节数
	if w.fc != nil {
		w.fc(int64(n))
	}
	return
}
