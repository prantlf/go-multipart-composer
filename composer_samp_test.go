package composer_test

import (
	"fmt"
	"log"
	"os"
	"strings"

	composer "github.com/prantlf/go-multipart-composer"
	"github.com/prantlf/go-multipart-composer/demo"
)

func Example() {
	// Create a new multipart message composer with a random boundary.
	comp := composer.NewComposer()
	// Close added files or readers if a failure before DetachReader occurred.
	// Not needed if you add no file, or if you add or just one file and then
	// do not abandon the composer before you succeed to return the result of
	// DetachReader or DetachReaderWithSize.
	defer comp.Close()

	// Add a textual field.
	comp.AddField("comment", "a comment")
	// Add a file content. Fails if the file cannot be opened.
	if err := comp.AddFile("file", "test.txt"); err != nil {
		log.Fatal(err)
	}

	// Get the content type of the composed multipart message.
	contentType := comp.FormDataContentType()
	// Collect the readers for added fields and files to a single compound
	// reader including the total size and empty the composer by detaching
	// the original readers from it.
	reqBody, contentLength, err := comp.DetachReaderWithSize()
	if err != nil {
		log.Fatal(err)
	}
	// Close added files or readers after the request body reader was used.
	// Not needed if the consumer of reqBody is called right away and will
	// guarantee to close the reader even in case of failure. Because this
	// is the case here, here it is for demonstration purposes only.
	defer reqBody.Close()

	// Make a network request with the composed content type and request body.
	demo.PrintRequestWithLength(contentLength, contentType, reqBody)
	// Output:
	// Content-Length: 383
	// Content-Type: multipart/form-data; boundary=1879bcd06ac39a4d8fa5
	//
	// --1879bcd06ac39a4d8fa5
	// Content-Disposition: form-data; name="comment"
	//
	// a comment
	// --1879bcd06ac39a4d8fa5
	// Content-Disposition: form-data; name="file"; filename="test.txt"
	// Content-Type: text/plain; charset=utf-8
	//
	// text file content
	// --1879bcd06ac39a4d8fa5--
}

func ExampleComposer() {
	// Create an invalid composer for results returned in case of error.
	comp := composer.Composer{}

	fmt.Printf("Empty composer: %v", comp.Boundary() == "")
	// Output:
	// Empty composer: true
}

func ExampleNewComposer() {
	// Create a new multipart message composer with a random boundary.
	comp := composer.NewComposer()

	fmt.Printf("Close added files or readers: %v", comp.CloseReaders)
	// Output:
	// Close added files or readers: true
}

func ExampleComposer_Boundary() {
	comp := composer.NewComposer()

	// Get the initial randomly-genenrated boundary.
	boundary := comp.Boundary()

	fmt.Printf("Boundary set: %v", len(boundary) > 0)
	// Output:
	// Boundary set: true
}

func ExampleComposer_SetBoundary() {
	comp := composer.NewComposer()

	// Set an explicit boundary to separate the message parts.
	comp.SetBoundary("3a494cd3b73de6555202")

	fmt.Print(comp.Boundary())
	// Output:
	// 3a494cd3b73de6555202
}

func ExampleComposer_ResetBoundary() {
	comp := composer.NewComposer()
	comp.SetBoundary("1")

	// Generate a new random boundary to separate the message parts.
	comp.ResetBoundary()

	fmt.Printf("Boundary reset: %v", len(comp.Boundary()) > 1)
	// Output:
	// Boundary reset: true
}

func ExampleComposer_FormDataContentType() {
	comp := composer.NewComposer()

	// Get the content type for the composed multipart message.
	contentType := comp.FormDataContentType()

	demo.PrintContentType(contentType)
	// Output:
	// Content-Type: multipart/form-data; boundary=1879bcd06ac39a4d8fa5
}

func ExampleComposer_AddField() {
	comp := composer.NewComposer()

	// Add a textual field.
	comp.AddField("foo", "bar")

	demo.PrintRequestBody(comp.DetachReader())
	// Output:
	// --1879bcd06ac39a4d8fa5
	// Content-Disposition: form-data; name="foo"
	//
	// bar
	// --1879bcd06ac39a4d8fa5--
}

func ExampleComposer_AddFieldReader() {
	comp := composer.NewComposer()

	// Add a textual field with a value supplied by a reader.
	comp.AddFieldReader("foo", strings.NewReader("bar"))

	demo.PrintRequestBody(comp.DetachReader())
	// Output:
	// --1879bcd06ac39a4d8fa5
	// Content-Disposition: form-data; name="foo"
	//
	// bar
	// --1879bcd06ac39a4d8fa5--
}

func ExampleComposer_AddFile() {
	comp := composer.NewComposer()

	// Add a file content. Fails if the file cannot be opened.
	if err := comp.AddFile("file", "test.txt"); err != nil {
		log.Fatal(err)
	}

	demo.PrintRequestBody(comp.DetachReader())
	// Output:
	// --1879bcd06ac39a4d8fa5
	// Content-Disposition: form-data; name="file"; filename="test.txt"
	// Content-Type: text/plain; charset=utf-8
	//
	// text file content
	// --1879bcd06ac39a4d8fa5--
}

func ExampleComposer_AddFileReader() {
	comp := composer.NewComposer()

	// Add a file content supplied as a separate reader.
	file, err := os.Open("test.txt")
	if err != nil {
		log.Fatal(err)
	}
	comp.AddFileReader("file", "test.txt", file)

	demo.PrintRequestBody(comp.DetachReader())
	// Output:
	// --1879bcd06ac39a4d8fa5
	// Content-Disposition: form-data; name="file"; filename="test.txt"
	// Content-Type: text/plain; charset=utf-8
	//
	// text file content
	// --1879bcd06ac39a4d8fa5--
}

func ExampleComposer_DetachReader() {
	comp := composer.NewComposer()

	// Get a multipart message with no parts.
	reqBody := comp.DetachReader()

	demo.PrintRequestBody(reqBody)
	// Output:
	// --1879bcd06ac39a4d8fa5--
}

func ExampleComposer_DetachReaderWithSize() {
	comp := composer.NewComposer()

	// Get a multipart message with no parts including its length.
	reqBody, contentLength, err := comp.DetachReaderWithSize()
	if err != nil {
		log.Fatal(err)
	}

	demo.PrintContentLength(contentLength)
	demo.PrintContentType(comp.FormDataContentType())
	demo.PrintRequestBody(reqBody)
	// Output:
	// Content-Length: 68
	// Content-Type: multipart/form-data; boundary=1879bcd06ac39a4d8fa5
	//
	// --1879bcd06ac39a4d8fa5--
}

func ExampleComposer_Clear() {
	comp := composer.NewComposer()
	comp.AddField("foo", "bar")

	// Abandon the composed content and clear the added fields.
	comp.Clear()

	comp.AddField("foo", "bar")

	demo.PrintRequestBody(comp.DetachReader())
	// Output:
	// --1879bcd06ac39a4d8fa5
	// Content-Disposition: form-data; name="foo"
	//
	// bar
	// --1879bcd06ac39a4d8fa5--
}

func ExampleComposer_Close() {
	comp := composer.NewComposer()

	// Add a file reader which will be closed automatically.
	file, err := os.Open("test.txt")
	if err != nil {
		log.Fatal(err)
	}
	comp.AddFileReader("file", "test.txt", file)

	// Close the added files and readers.
	comp.Close()
	if _, err := file.Stat(); err == nil {
		log.Fatal("open")
	}

	// Start again with disabled closing of files and readers.
	comp.Clear()
	comp.CloseReaders = false

	// Add a file reader which will not be closed automatically.
	file, err = os.Open("test.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	comp.AddFileReader("file", "test.txt", file)

	// Adding a file by path is impossible if automatic closing is disabled.
	if err := comp.AddFile("file", "test.txt"); err == nil {
		log.Fatal("added")
	}

	// Getting the final reader or closing the composer will not close the file.
	reqBody := comp.DetachReader()
	comp.Close()
	if _, err := file.Stat(); err != nil {
		log.Fatal(err)
	}

	demo.PrintRequest(comp.FormDataContentType(), reqBody)
	// Output:
	// Content-Type: multipart/form-data; boundary=1879bcd06ac39a4d8fa5
	//
	// --1879bcd06ac39a4d8fa5
	// Content-Disposition: form-data; name="file"; filename="test.txt"
	// Content-Type: text/plain; charset=utf-8
	//
	// text file content
	// --1879bcd06ac39a4d8fa5--
}
