# 多协程并发下载
```

package main

import (
	"context"
	"fmt"
	"time"
	"github.com/s84662355/bigfiledownloader"
)

func main() {
	// 创建带超时的上下文（超时时间5分钟）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel() // 操作完成后取消上下文，释放资源

	// 初始化下载器（5个并发线程，进度回调函数打印下载进度）
	downloader := bigfiledownloader.NewBigDownloader(5, func(progress float64) {
		fmt.Printf("当前下载进度：%.2f%%\n", progress*100) // 格式化输出进度百分比
	})

	// 执行下载任务
	err := downloader.Download(
		ctx,
		"https://sytg-souncs.com/Ctrrsion/test/updatm6.66.zip", // 原始文件URL
		"AAAAA.zip",                                           // 保存的文件名
	)

	// 处理下载结果
	if err != nil {
		fmt.Printf("下载失败：%v\n", err)
	} else {
		fmt.Println("文件下载完成")
	}
}

```

 
