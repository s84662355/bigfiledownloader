# bigfiledownloader 示例
### ## #大文件分片下载
 ```go
package main

import (
	"github.com/s84662355/bigfiledownloader"
	"log"
	 
	 
)

func main(){
    /// 分片数量  进度函数
 	err := bigfiledownloader.NewBigDownloader(38, func(d  float64) {
		log.Printf("下载进度: %.2f%%",100*d) 
	}).Download(`https://sytxxxxxxxxxxxxxxxyuncs.com/Ctrxxxxxxxsion/test/upxxxx6.66.zip`, "E:\\bigfiledownloader\\Download\\bigfiledownloader.zip")
   	log.Println(err)
}
```