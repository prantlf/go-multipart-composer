// Package composer prepares bodies of HTTP requests with MIME multipart
// messages without reading entire file contents to memory. Instead of writing
// files to multipart Writer right away, it collects Readers for each part
// of the form and lets them stream to the network once the request has been
// sent. Avoids buffering of the request body simpler than with goroutines
// and pipes.
//
// Text fields and files can be appended by convenience methods:
//
//     comp := composer.NewComposer()
//     comp.AddField("comment", "a comment")
//     err := comp.AddFile("file", "test.txt")
//
// The multipart form-data content type and a reader for the full request
// body can be passed directly the HTTP request methods. They close a
// closable writer even in case of failure:
//
//     resp, err := http.DefaultClient.Post("http://host.com/upload",
//       comp.FormDataContentType(), comp.DetachReader())
package composer

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/textproto"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/prantlf/go-sizeio"
)

// A Composer generates multipart messages with delayed content supplied
// by readers.
type Composer struct {
	// CloseReaders, if set to false, prevents closing of added files
	// or readers when Close is called, or when the reader returned by
	// DetachReader is closed. The initial value set by NewComposer is true.
	CloseReaders bool

	boundary string
	readers  []io.Reader
}

// NewComposer returns a new multipart message Composer with a random
// boundary.
//
// If you are going to add parts with readers that needs closing (files),
// defer a call to Close in case an error occurs, the best right after
// calling this method.
func NewComposer() *Composer {
	return &Composer{boundary: randomBoundary(), CloseReaders: true}
}

// Boundary returns the Composer's boundary.
func (c *Composer) Boundary() string {
	return c.boundary
}

// SetBoundary overrides the Composer's initial boundary separator
// with an explicit value.
//
// SetBoundary must be called before any parts are added, or after all
// parts were detached by one of the DetachReader methods. may only
// contain certain ASCII characters, and must be non-empty and
// at most 70 bytes long. (See RFC 2046, section 5.1.1.)
func (c *Composer) SetBoundary(boundary string) error {
	if len(c.readers) > 0 {
		return errors.New("multipart: SetBoundary called after add")
	}
	// rfc2046#section-5.1.1
	if len(boundary) < 1 || len(boundary) > 70 {
		return errors.New("multipart: invalid boundary length")
	}
	end := len(boundary) - 1
	for i, c := range boundary {
		if 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' {
			continue
		}
		switch c {
		case '\'', '(', ')', '+', '_', ',', '-', '.', '/', ':', '=', '?':
			continue
		case ' ':
			if i != end {
				continue
			}
		}
		return errors.New("multipart: invalid boundary character")
	}
	c.boundary = boundary
	return nil
}

// ResetBoundary overrides the Composer's current boundary separator
// with a randomly generared one.
//
// ResetBoundary must be called before any parts are added, or after all
// parts were detached by one of the DetachReader methods.
func (c *Composer) ResetBoundary() error {
	if len(c.readers) > 0 {
		return errors.New("multipart: RandomizeBoundary called after add")
	}
	c.boundary = randomBoundary()
	return nil
}

// FormDataContentType returns the value of Content-Type for an HTTP request
// with the body prepared by this Composer. It will include the constant
// "multipart/form-data" and this Composers's Boundary.
func (c *Composer) FormDataContentType() string {
	boundary := c.boundary
	// Quote the boundary if it contains any of the special characters
	// defined by RFC 2045, or space.
	if strings.ContainsAny(boundary, `()<>@,;:\"/[]?= `) {
		boundary = `"` + boundary + `"`
	}
	return "multipart/form-data; boundary=" + boundary
}

// CreateFilePart creates a new general multipart section, but does not add
// it to the composer yet.
// Passing the returned header to AddPart will add it to the composer.
func (c *Composer) CreatePart(disposition map[string]string) textproto.MIMEHeader {
	head := make(textproto.MIMEHeader)
	var buf bytes.Buffer
	fmt.Fprint(&buf, "form-data")
	for key, val := range disposition {
		fmt.Fprintf(&buf, `; %s="%s"`, key, escapeQuotes(val))
	}
	head.Set("Content-Disposition", buf.String())
	return head
}

// CreateFilePart creates a new multipart section for a field, but does not add
// it to the composer yet.
// Passing the returned header to AddPart will add it to the composer.
func (c *Composer) CreateFieldPart(name string) textproto.MIMEHeader {
	head := make(textproto.MIMEHeader)
	head.Set("Content-Disposition", fmt.Sprintf(
		"form-data; name=\"%s\"", escapeQuotes(name)))
	return head
}

// CreateFilePart creates a new multipart section for a file, but does not add
// it to the composer yet.
// Passing the returned header to AddPart will add it to the composer.
func (c *Composer) CreateFilePart(fieldName, fileName string) textproto.MIMEHeader {
	head := make(textproto.MIMEHeader)
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	head.Set("Content-Disposition", fmt.Sprintf(
		"form-data; name=\"%s\"; filename=\"%s\"", escapeQuotes(fieldName), escapeQuotes(fileName)))
	head.Set("Content-Type", contentType)
	return head
}

// AddPart creates a new multipart section prepared earlier with CreatePart,
// CreateFieldPart or CreateFilePart.
// It inserts all headers prepared earlier and then appends the value reader.
func (c *Composer) AddPart(header textproto.MIMEHeader, reader io.Reader) {
	var buf bytes.Buffer
	var delimiter string
	if len(c.readers) > 0 {
		delimiter = "\r\n"
	}
	fmt.Fprintf(&buf, "%s--%s\r\n", delimiter, c.boundary)
	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, val := range header[key] {
			fmt.Fprintf(&buf, "%s: %s\r\n", key, val)
		}
	}
	fmt.Fprintf(&buf, "\r\n")
	c.readers = append(c.readers, bytes.NewReader(buf.Bytes()), reader)
}

// AddField creates a new multipart section with a field value.
// It inserts a header with the provided field name and value.
func (c *Composer) AddField(name, value string) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\n\r\n%s",
		c.delimiter(), c.boundary, escapeQuotes(name), value)
	c.readers = append(c.readers, bytes.NewReader(buf.Bytes()))
}

// AddFieldReader creates a new multipart section with a field value.
// It inserts a header using the given field name and then appends
// the value reader.
func (c *Composer) AddFieldReader(name string, reader io.Reader) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\n\r\n",
		c.delimiter(), c.boundary, escapeQuotes(name))
	c.readers = append(c.readers, bytes.NewReader(buf.Bytes()), reader)
}

// AddFile is a convenience wrapper around AddFileReader. It opens the given
// file and uses its name, stats and content to create the new part.
//
// The opened file wil be owned by the Composer. Do not forget to close
// the composer, once you do not need it, or defer the closure to perform
// it automatically in case of a failure.
func (c *Composer) AddFile(fieldName, filePath string) error {
	if !c.CloseReaders {
		return errors.New("multipart: adding file by path forbidden")
	}
	reader, err := sizeio.OpenFile(filePath)
	if err != nil {
		return err
	}
	c.AddFileReader(fieldName, filepath.Base(filePath), reader)
	return nil
}

// AddFileObject is a convenience wrapper around AddFileReader. It uses
// the name, stats and content of the opened file to create the new part.
//
// The opened file wil be owned by the Composer. Do not forget to close
// the composer, once you do not need it, or defer the closure to perform
// it automatically in case of a failure. However, do not close the source
// file. The reader taking part in the request body creation would fail.
func (c *Composer) AddFileObject(fieldName string, file *os.File) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	c.AddFileReader(fieldName, stat.Name(), sizeio.SizeReadCloser(file, stat.Size()))
	return nil
}

// AddFileReader creates a new multipart section with a file content.
// It inserts a header using the given field name, file name and the content
// type inferred from the file extension, then appends the reader's content.
//
// If the reader passed in is a ReaderCloser, it will be owned and eventually
// freed by the Composer. Do not forget to close the composer, once you do
// not need it, or defer the closure to perform it automatically in case of
// a failure. However, do not close the source file. The reader taking part
// in the request body creation would fail.
func (c *Composer) AddFileReader(fieldName, fileName string, reader io.Reader) {
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s--%s\r\nContent-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\nContent-Type: %s\r\n\r\n",
		c.delimiter(), c.boundary, escapeQuotes(fieldName), escapeQuotes(fileName), contentType)
	c.readers = append(c.readers, bytes.NewReader(buf.Bytes()), reader)
}

// DetachReader finishes the multipart message by adding the trailing
// boundary end line to the output and moves the closable readers to be
// closed with the returned compound reader.
func (c *Composer) DetachReader() io.ReadCloser {
	c.appendLastBoundary()
	return c.detachReader()
}

// DetachReaderWithSize finishes the multipart message by adding the trailing
// boundary end line to the output and moves the closable readers to be
// closed with the returned compound reader. It tries computing the total
// request body size, which will work if size was available for all readers.
//
// If it fails, the composer instance will not be closed.
func (c *Composer) DetachReaderWithSize() (io.ReadCloser, int64, error) {
	c.appendLastBoundary()
	size, err := c.totalSize()
	if err != nil {
		return nil, 0, err
	}
	allReader := c.detachReader()
	return allReader, size, nil
}

// Clear closes all closable readers added by AddFileReader or AddFile and
// clears their collection, making the composer ready to start empty again.
func (c *Composer) Clear() {
	c.Close()
	c.readers = nil
}

// Close closes all closable readers added by AddFileReader or AddFile.
// If some of them fail, the first error will be returned.
func (c *Composer) Close() error {
	if c.CloseReaders {
		return closeAll(c.readers)
	}
	return nil
}

type composedReader struct {
	io.Reader
	readers []io.Reader
}

func (r composedReader) Close() error {
	return closeAll(r.readers)
}

func (c *Composer) totalSize() (int64, error) {
	var size int64
	for _, reader := range c.readers {
		if withSize, ok := reader.(sizeio.WithSize); ok {
			size += withSize.Size()
		} else {
			return 0, errors.New("multipart: reader without size encountered")
		}
	}
	return size, nil
}

func (c *Composer) detachReader() io.ReadCloser {
	var readers []io.Reader
	if c.CloseReaders {
		readers = c.readers
	}
	allReader := composedReader{io.MultiReader(c.readers...), readers}
	c.readers = nil
	return allReader
}

func closeAll(readers []io.Reader) error {
	var firstErr error
	for _, reader := range readers {
		if closer, ok := reader.(io.ReadCloser); ok {
			err := closer.Close()
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (c *Composer) appendLastBoundary() {
	c.readers = append(c.readers,
		strings.NewReader(fmt.Sprintf("\r\n--%s--\r\n", c.boundary)))
}

func (c *Composer) delimiter() string {
	if len(c.readers) > 0 {
		return "\r\n"
	}
	return ""
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(value string) string {
	return quoteEscaper.Replace(value)
}

func randomBoundary() string {
	var buf [30]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}
