#多协程并发下载
```

	ctx, _ := context.WithTimeout(context.Background(), 5*60*time.Second)
	err := NewBigDownloader(5, func(ddd float64) {
		fmt.Println(ddd)
	}).Download(
		ctx, 
		"https://sytg-souncs.com/Ctrrsion/test/updatm6.66.zip", 
		"1111sa.zip",
	)
	fmt.Println(err)

```

 