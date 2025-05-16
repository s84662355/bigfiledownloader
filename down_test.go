package bigfiledownloader

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// go test   -run TestDown 
func TestDown(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 5*60*time.Second)
	err := NewBigDownloader(5, func(ddd float64) {
		fmt.Println(ddd)
	}).Download(ctx, `https://sytg-brow.aliyuncs.com/Ctrln/test/updateProgram6.66.zip`, "1111sa.zip")
	fmt.Println(err)
}

// go test -run TestDownPng 
func TestDownPng(t *testing.T) {
	err := NewBigDownloader(2, func(ddd float64) {
		fmt.Println(ddd)
	}).Download(context.Background(), `https://csdnimg.cn/release/blogv2/dist/pc/img/reprint.png`, "reprint.png")
	fmt.Println(err)
}
