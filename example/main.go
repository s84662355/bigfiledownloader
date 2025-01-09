package main

import (
	"log"
	"os"

	"github.com/s84662355/bigfiledownloader"
)

func main() {
	os.Mkdir("E:\\bigfiledownloader\\Download", 0o777)

	err := bigfiledownloader.NewBigDownloader(38, func(d float64) {
		log.Printf("下载进度: %.2f%%", d*100)
	}).Download(`https://sytg-bt-1.aliycs.com/ghjion/test/updagram6.66.zip`, "E:\\bigfiledownloader\\Download\\bigfiledownloader.zip")
	log.Println(err)
}
