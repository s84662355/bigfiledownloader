package main

import (
	"github.com/s84662355/bigfiledownloader"
	"log"
	"os"
	 
)

func main(){
	os.Mkdir("E:\\bigfiledownloader\\Download", 0o777)

	ddd:=bigfiledownloader.NewBigDownloader(38, func(d  float64) {
		log.Printf("下载进度: %.2f%%",100*d) 
	}) 



 	err := ddd.Download(`https://sytg-xxxxxxxxxxxxxxxom/Ctrxxxversion/testxxogram6.66.zip`, "E:\\bigfiledownloader\\Download\\bigfiledownloader.zip")
   	log.Println(err) 
 	err = ddd.Download(`https://sytxxxxxxxcom/Ctxxxxxxsion/test/updxxam6.66.zip`, "E:\\bigfiledownloader\\Download\\bigfixxxxder.zip")
   	log.Println(err)

}