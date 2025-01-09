package main

import (
	"log"

	"github.com/s84662355/bigfiledownloader"
)

func main() {
	ddd := bigfiledownloader.NewBigDownloader(48, func(d float64) {
		log.Printf("下载进度: %.2f%%", 100*d)
	})

	err := ddd.Download(`https://sytg-browser.oss-ap-southeast-1.aliyuncs.com/CtrlFire-version/test/updateProgram6.66.zip`, "G:\\work\\bigfiledownloader\\example\\bigfiledownloader.zip")
	log.Println(err)
}
