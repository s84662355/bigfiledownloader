# bigfiledownloader 示例
### ## #大文件分片下载
 ```go
package main

import (
	"github.com/s84662355/bigfiledownloader"
	"log"
	"os" 
)

func main(){
	os.Mkdir("E:\\bigfilxxxxxxxxxedownloader\\Download", 0o777)
 	err := bigfiledownloader.NewBigDownloader(38, func(d  float64) {
			log.Printf("下载进度: %.2f%%",d) 
	}).Download(`https://xxx.xxx.xx.com/Ctsion/test/xxxx.zip`, "E:\\xxxxxxxxx\\Download\\xxxxx.zip")
   	log.Println(err)
}
```