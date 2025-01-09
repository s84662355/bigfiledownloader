package main

import (
	"log"
	"github.com/s84662355/bigfiledownloader"
)

func main() {
	/// 分片数量  进度函数
	ddd := bigfiledownloader.NewBigDownloader(38, func(d float64) {
		log.Printf("下载进度: %.2f%%", 100*d)
	})

	err := ddd.Download(`https://sytg-bxxxxxxxxxxxxncs.com/Ctxxxxxxxxxxxxxxion/test/upxxxxxxxxxxx.66.zip`, "G:\\work\\bigfiledownloader\\example\\bigfiledownloader.zip")
	log.Println(err)
}
