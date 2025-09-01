package bigfiledownloader

import (
	"errors"
)

// 自定义错误变量定义
var (
	ErrDownloading          = errors.New("download is in progress")
	ErrContentLengthZero    = errors.New("content length is zero")
	ErrMissingAcceptRanges  = errors.New("request failed or missing Accept-Ranges header")
	ErrOpenFileFailed       = errors.New("failed to open file")
	ErrFileTruncateFailed   = errors.New("failed to truncate file")
	ErrInvalidRange         = errors.New("invalid range: start >= end")
	ErrCreateConnectionFailed = errors.New("failed to create connection")
	ErrDownloadPartialFailed = errors.New("failed to download partial content")
)


// 新增的自定义错误变量
var (
	ErrCreateRequestFailed = errors.New("failed to create HTTP request")
	ErrRequestFailed       = errors.New("HTTP request failed")
	ErrReadTimeout         = errors.New("data read timeout")
	ErrContextTimeout      = errors.New("context timeout during read/write")
)