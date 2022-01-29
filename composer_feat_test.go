package composer_test

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	composer "github.com/prantlf/go-multipart-composer"
)

func TestComposer_SetBoundary_simple(t *testing.T) {
	comp := composer.NewComposer()
	if err := comp.SetBoundary("foo"); err != nil {
		t.Error("composer: simple failed -", err)
	}
	if comp.Boundary() != "foo" {
		t.Error("composer: simple not set")
	}
}

func TestComposer_SetBoundary_late(t *testing.T) {
	comp := composer.NewComposer()
	comp.AddField("foo", "bar")
	if err := comp.SetBoundary("foo"); err == nil {
		t.Error("composer: late succeeded")
	}
}

func TestComposer_SetBoundary_empty(t *testing.T) {
	comp := composer.NewComposer()
	if err := comp.SetBoundary(""); err == nil {
		t.Error("composer: empty succeeded")
	}
}

func TestComposer_SetBoundary_long(t *testing.T) {
	comp := composer.NewComposer()
	if err := comp.SetBoundary("01234567890123456789012345678901234567890123456789012345678901234567890123456789"); err == nil {
		t.Error("composer: long succeeded")
	}
}

func TestComposer_SetBoundary_invalid(t *testing.T) {
	comp := composer.NewComposer()
	if err := comp.SetBoundary(" "); err == nil {
		t.Error("composer: invalid succeeded")
	}
}

func TestComposer_SetBoundary_special(t *testing.T) {
	comp := composer.NewComposer()
	if err := comp.SetBoundary("foo / bar"); err != nil {
		t.Error("composer: special failed -", err)
	}
	if !strings.HasSuffix(comp.FormDataContentType(), `"foo / bar"`) {
		t.Error("composer: special not quoted")
	}
}

func TestComposer_ResetBoundary(t *testing.T) {
	comp := composer.NewComposer()
	comp.SetBoundary("foo")
	if err := comp.ResetBoundary(); err != nil {
		t.Error("composer: reset failed -", err)
	}
	if comp.Boundary() == "foo" {
		t.Error("composer: not reset")
	}
}

func TestComposer_ResetBoundary_late(t *testing.T) {
	comp := composer.NewComposer()
	comp.AddField("foo", "bar")
	if err := comp.ResetBoundary(); err == nil {
		t.Error("composer: late succeeded")
	}
}

func TestComposer_AddFile_missing(t *testing.T) {
	comp := composer.NewComposer()
	if err := comp.AddFile("file", "missing.txt"); err == nil {
		t.Error("composer: invalid file added")
	}
}

func TestComposer_AddFile_text(t *testing.T) {
	comp := composer.NewComposer()
	comp.AddFile("file", "demo/test.txt")
	out, _ := ioutil.ReadAll(comp.DetachReader())
	if !strings.Contains(string(out), "Content-Type: text/plain") {
		t.Error("composer: unrecognised text")
	}
}

func TestComposer_AddFile_binary(t *testing.T) {
	comp := composer.NewComposer()
	comp.AddFile("file", "demo/test.bin")
	out, _ := ioutil.ReadAll(comp.DetachReader())
	if !strings.Contains(string(out), "Content-Type: application/octet-stream") {
		t.Error("composer: unrecognised binary")
	}
}

func TestComposer_AddFileObject_opened(t *testing.T) {
	comp := composer.NewComposer()
	file, _ := os.Open("demo/test.bin")
	defer file.Close()
	comp.AddFileObject("file", file)
	out, _ := ioutil.ReadAll(comp.DetachReader())
	if !strings.Contains(string(out), "Content-Type: application/octet-stream") {
		t.Error("composer: unrecognised file object")
	}
}

func TestComposer_AddFileObject_closed(t *testing.T) {
	comp := composer.NewComposer()
	file, _ := os.Open("demo/test.bin")
	file.Close()
	if err := comp.AddFileObject("file", file); err == nil {
		t.Error("composer: closed file object added")
	}
}

func TestComposer_DetachReaderWithSize_nosize(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		_, err := pipeWriter.Write([]byte{42})
		pipeWriter.CloseWithError(err)
	}()
	comp := composer.NewComposer()
	comp.AddFieldReader("foo", pipeReader)
	out, size, err := comp.DetachReaderWithSize()
	if err == nil {
		t.Error("composer: reader without size accepted")
	}
	if size != 0 {
		t.Error("composer: invalid size not zero")
	}
	if out != nil {
		t.Error("composer: invalid reader not nil")
	}
}

func TestComposer_AddField_plain(t *testing.T) {
	comp := composer.NewComposer()
	comp.AddField("name", "demo/test.bin")
	out, _ := ioutil.ReadAll(comp.DetachReader())
	if !strings.Contains(string(out), "demo/test.bin") {
		t.Error("composer: missing field")
	}
}

func TestComposer_AddField_escaped(t *testing.T) {
	comp := composer.NewComposer()
	comp.AddField("name \"a\"", "demo/test.bin")
	out, _ := ioutil.ReadAll(comp.DetachReader())
	if !strings.Contains(string(out), "name=\"name \\\"a\\\"") {
		t.Error("composer: unescaped field")
	}
}

func TestComposer_AddPart_file(t *testing.T) {
	comp := composer.NewComposer()
	part := comp.CreateFilePart("file", "my")
	comp.AddPart(part, strings.NewReader("test"))
	out, _ := ioutil.ReadAll(comp.DetachReader())
	if !strings.Contains(string(out), "name=\"file\"") ||
		!strings.Contains(string(out), "filename=\"my\"") ||
		!strings.Contains(string(out), "test") {
		t.Error("composer: part file not added")
	}
}

func TestComposer_AddPart_field(t *testing.T) {
	comp := composer.NewComposer()
	part := comp.CreateFieldPart("field")
	comp.AddPart(part, strings.NewReader("test"))
	out, _ := ioutil.ReadAll(comp.DetachReader())
	if !strings.Contains(string(out), "name=\"field\"") ||
		!strings.Contains(string(out), "test") {
		t.Error("composer: part field not added")
	}
}

func TestComposer_AddPart_part(t *testing.T) {
	comp := composer.NewComposer()
	disp := make(map[string]string)
	disp["name"] = "value"
	part := comp.CreatePart(disp)
	comp.AddPart(part, strings.NewReader("test"))
	out, _ := ioutil.ReadAll(comp.DetachReader())
	println(string(out))
	if !strings.Contains(string(out), "name=\"value\"") ||
		!strings.Contains(string(out), "test") {
		t.Error("composer: part not added")
	}
}

func TestComposer_AddPart_2parts(t *testing.T) {
	comp := composer.NewComposer()
	disp := make(map[string]string)
	disp["name"] = "value1"
	part := comp.CreatePart(disp)
	comp.AddPart(part, strings.NewReader("test1"))
	disp = make(map[string]string)
	disp["name"] = "value2"
	part = comp.CreatePart(disp)
	comp.AddPart(part, strings.NewReader("test2"))
	out, _ := ioutil.ReadAll(comp.DetachReader())
	println(string(out))
	if !strings.Contains(string(out), "name=\"value1\"") ||
		!strings.Contains(string(out), "test1") ||
		!strings.Contains(string(out), "name=\"value2\"") ||
		!strings.Contains(string(out), "test2") {
		t.Error("composer: 2 parts not added")
	}
}
