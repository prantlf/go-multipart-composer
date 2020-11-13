# go-multipart-composer

Prepares bodies of HTTP requests with MIME multipart messages according to [RFC7578] without reading entire file contents to memory. Instead of writing files to a [multipart writer] right away, it collects [readers] for each part of the form and lets them stream to the network once the request has been sent. Avoids buffering of the request body simpler than with [goroutines] and [pipes]. See the [documentation] for more information.

## Installation

Add this package to `go.mod` and `go.sub` in your Go project:

    go get github.com/prantlf/go-multipart-composer

## Usage

Upload a file with comment:

```go
import (
	"net/http"
	"github.com/prantlf/go-multipart-composer"
)
// compose a multipart form-data content
comp := composer.NewComposer()
comp.AddField("comment", "a comment")
err := comp.AddFile("file", "test.txt")
// post a request with the generated content type and body
resp, err := http.DefaultClient.Post("http://host.com/upload",
  comp.FormDataContentType(), comp.DetachReader())
```

If the server does not support chunked encoding and requires `Content-=Length` in the header:

```go
comp := composer.NewComposer()
comp.AddField("comment", "a comment")
err := comp.AddFile("file", "test.txt")
reqBody, contentLength, err := comp.DetachReaderWithSize()
if err != nil {
  comp.Close() // DetachReaderWithSize does not close the composer on failure
  log.Fatal(err)
}
// post a request with the generated body, content type and content length
req, err := http.NewRequest("POST", "http://host.com/upload", reqBody)
req.Header.Add("Content-Type", comp.FormDataContentType())
req.ContentLength = contentLength
resp, err := http.DefaultClient.Do(request)
```

See the [documentation] for the full interface.

[documentation]: https://pkg.go.dev/github.com/prantlf/go-multipart-composer
[readers]: https://golang.org/pkg/io/#Reader
[multipart writer]: https://golang.org/pkg/mime/multipart/#Writer
[goroutines]: https://tour.golang.org/concurrency/1
[pipes]: https://golang.org/pkg/io/#Pipe
[RFC7578]: https://tools.ietf.org/html/rfc7578
