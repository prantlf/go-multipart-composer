package demo

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
)

const fixedBoundary = "3a494cd3b73de6555202"
const commonBoundary = "1879bcd06ac39a4d8fa5"

var contentTypeBoundary = regexp.MustCompile("boundary=.+")
var requestBodyBoundary = regexp.MustCompile("--[0-9a-z]+")

func PrintRequestWithLength(contentLength int64, contentType string, reqBody io.ReadCloser) {
	PrintContentLength(contentLength)
	PrintRequest(contentType, reqBody)
}

func PrintRequest(contentType string, reqBody io.ReadCloser) {
	PrintContentType(contentType)
	fmt.Println()
	PrintRequestBody(reqBody)
}

func PrintContentType(contentType string) {
	if contentType[len(contentType)-20:] != fixedBoundary {
		contentType = contentTypeBoundary.ReplaceAllLiteralString(contentType, "boundary="+commonBoundary)
	}
	fmt.Printf("Content-Type: %s\n", contentType)
}

func PrintContentLength(contentLength int64) {
	fmt.Printf("Content-Length: %d\n", contentLength)
}

func PrintRequestBody(reqBody io.ReadCloser) {
	defer reqBody.Close()
	reqBuf := stringifyReader(reqBody)
	if err := reqBody.Close(); err != nil {
		log.Fatal(err)
	}
	if reqBuf[len(reqBuf)-24:len(reqBuf)-4] != fixedBoundary {
		reqBuf = requestBodyBoundary.ReplaceAllLiteralString(reqBuf, "--"+commonBoundary)
	}
	fmt.Println(reqBuf)
}

func stringifyReader(reqBody io.Reader) string {
	builder := new(strings.Builder)
	if _, err := io.Copy(builder, reqBody); err != nil {
		log.Fatal(err)
	}
	return strings.ReplaceAll(builder.String(), "\r\n", "\n")
}
