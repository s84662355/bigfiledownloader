package main

import (
	"github.com/s84662355/bigfiledownloader"
	"log"
	"os"
	 
)

func main(){
	os.Mkdir("E:\\bigfiledownloader\\Download", 0o777)
 	err := bigfiledownloader.NewBigDownloader(38, func(d  float64) {
 
			log.Printf("下载进度: %.2f%%",d) 
	}).Download(`https://sytg-browser.oss-ap-southeast-1.aliyuncs.com/CtrlFire-version/test/updateProgram6.66.zip`, "E:\\bigfiledownloader\\Download\\bigfiledownloader.zip")
   	log.Println(err)
}