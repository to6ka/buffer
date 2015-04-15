package buffer

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"io"
	"io/ioutil"
	"math"
	"os"
	"testing"
)

func BenchmarkMemory(b *testing.B) {
	buf := New(32 * 1024)
	for i := 0; i < b.N; i++ {
		io.Copy(buf, io.LimitReader(rand.Reader, 32*1024))
		io.Copy(ioutil.Discard, buf)
	}
}

func TestPool(t *testing.T) {
	pool := NewPool(func() Buffer { return New(10) })
	buf := pool.Get()
	buf.Write([]byte("hello world"))
	pool.Put(buf)
}

func TestMemPool(t *testing.T) {
	pool := NewMemPool(10)
	buf := pool.Get()
	if n, err := buf.Write([]byte("hello world")); n != 10 {
		t.Errorf("wrote incorrect amount")
	} else if err == nil {
		t.Errorf("should have been a shortwrite error here")
	}
	pool.Put(buf)
	if buf.Len() > 0 {
		t.Errorf("should have emptied the buffer")
	}
}

func TestFilePool(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("there should have been a panic here!")
		}
	}()

	pool := NewFilePool(1024, "::~_bad_dir_~::")
	pool.Get()
}

func TestOverflow(t *testing.T) {
	buf := NewMulti(New(5), NewDiscard())
	buf.Write([]byte("Hello World"))

	data, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Error(err.Error())
	}

	if !bytes.Equal(data, []byte("Hello")) {
		t.Errorf("Expected Hello got %s", string(data))
	}

}

func TestWriteAt(t *testing.T) {
	var b BufferAt

	b = New(5)
	BufferAtTester(t, b)

	file, err := ioutil.TempFile("", "buffer")
	if err != nil {
		t.Error(err.Error())
	}
	defer os.Remove(file.Name())
	defer file.Close()

	b = NewFile(5, file)
	BufferAtTester(t, b)

}

func BufferAtTester(t *testing.T, b BufferAt) {
	b.WriteAt([]byte("abc"), 0)
	Compare(t, b, "abc")

	b.WriteAt([]byte("abc"), 1)
	Compare(t, b, "aabc")

	b.WriteAt([]byte("abc"), 2)
	Compare(t, b, "aaabc")

	b.WriteAt([]byte("abc"), 3)
	Compare(t, b, "aaaab")

	b.Read(make([]byte, 2))

	Compare(t, b, "aab")
	b.Reset()
}

func Compare(t *testing.T, b BufferAt, s string) {
	data := make([]byte, b.Len())
	n, _ := b.ReadAt(data, 0)
	if string(data[:n]) != s {
		t.Error("Mismatch:", string(data[:n]), s)
	}
}

func TestRing(t *testing.T) {
	ring := NewRing(New(3))
	if ring.Len() != 0 {
		t.Errorf("Ring non-empty start!")
	}

	if ring.Cap() != math.MaxInt64 {
		t.Errorf("Ring has < max capacity")
	}

	ring.Write([]byte("abc"))
	if ring.Len() != 3 {
		t.Errorf("expected ring len == 3")
	}

	ring.Write([]byte("de"))
	if ring.Len() != 3 {
		t.Errorf("expected riing len == 3")
	}

	data := make([]byte, 12)
	if n, err := ring.Read(data); err != nil {
		t.Error(err.Error())
	} else {
		if !bytes.Equal(data[:n], []byte("cde")) {
			t.Errorf("expected cde, got %s", data[:n])
		}
	}

	if ring.Len() != 0 {
		t.Errorf("ring should now be empty")
	}

	ring.Write([]byte("hello"))
	ring.Reset()

	if ring.Len() != 0 {
		t.Errorf("ring should still be empty")
	}
}

func TestGob(t *testing.T) {
	str := "HelloWorld"

	buf := NewUnboundedBuffer(2, 2)
	buf.Write([]byte(str))
	b := bytes.NewBuffer(nil)
	if err := gob.NewEncoder(b).Encode(&buf); err != nil {
		t.Error(err.Error())
		return
	}
	var buffer Buffer
	if err := gob.NewDecoder(b).Decode(&buffer); err != nil {
		t.Error(err.Error())
		return
	}
	data := make([]byte, len(str))
	if _, err := buffer.Read(data); err != nil && err != io.EOF {
		t.Error(err.Error())
	}
	if !bytes.Equal(data, []byte(str)) {
		t.Error("Gob Recover Failed... " + string(data))
	}
	buffer.Reset()
}

func TestDiscard(t *testing.T) {
	buf := NewDiscard()
	if buf.Cap() != math.MaxInt64 {
		t.Errorf("cap isn't infinite")
	}
	buf.Write([]byte("hello"))
	if buf.Len() != 0 {
		t.Errorf("buf should always be empty")
	}
}

func TestList(t *testing.T) {
	mem := New(10)
	ory := New(10)
	mem.Write([]byte("Hello"))
	ory.Write([]byte("world"))

	buf := List([]Buffer{mem, ory, NewDiscard()})
	if buf.Len() != 10 {
		t.Errorf("incorrect sum of lengths")
	}
	if buf.Cap() != math.MaxInt64 {
		t.Errorf("incorrect sum of caps")
	}

	buf.Reset()
	if buf.Len() != 0 {
		t.Errorf("buffer should be empty")
	}
}

func TestList2(t *testing.T) {
	mem := New(10)
	ory := New(10)
	mem.Write([]byte("Hello"))
	ory.Write([]byte("world"))

	buf := List([]Buffer{mem, ory})
	if buf.Len() != 10 {
		t.Errorf("incorrect sum of lengths")
	}
	if buf.Cap() != 20 {
		t.Errorf("incorrect sum of caps")
	}

	buf.Reset()
	if buf.Len() != 0 {
		t.Errorf("buffer should be empty")
	}
}

func TestSpill(t *testing.T) {
	buf := NewSpill(New(5), NewDiscard())
	buf.Write([]byte("Hello World"))
	data := make([]byte, 12)
	n, _ := buf.Read(data)
	if !bytes.Equal(data[:n], []byte("Hello")) {
		t.Error("ReadAt Failed. " + string(data[:n]))
	}
}

func TestSpill2(t *testing.T) {
	buf := NewSpill(New(5), nil)

	if buf.Cap() != math.MaxInt64 {
		t.Errorf("cap isn't infinite")
	}

	towrite := []byte("Hello World")
	m, _ := buf.Write(towrite)
	if m != len(towrite) {
		t.Errorf("failed to write all data: %d != %d", m, len(towrite))
	}
	data := make([]byte, 12)
	n, _ := buf.Read(data)
	if !bytes.Equal(data[:n], []byte("Hello")) {
		t.Error("Read Failed. " + string(data[:n]))
	}
}

func TestNoSpill(t *testing.T) {
	buf := NewSpill(New(1024), nil)
	buf.Write([]byte("Hello World"))
	data := make([]byte, 12)
	n, _ := buf.Read(data)
	if !bytes.Equal(data[:n], []byte("Hello World")) {
		t.Error("Read Failed. " + string(data[:n]))
	}
}

func TestFile(t *testing.T) {
	file, err := ioutil.TempFile("", "buffer")
	if err != nil {
		t.Error(err.Error())
	}
	defer os.Remove(file.Name())
	defer file.Close()

	buf := NewFile(1024, file)
	checkCap(t, buf, 1024)
	runPerfectSeries(t, buf)
	buf.Reset()

	buf = NewFile(3, file)
	buf.Write([]byte("abc"))
	buf.Read(make([]byte, 1))
	buf.Write([]byte("a"))
	d, _ := ioutil.ReadAll(buf)
	if !bytes.Equal(d, []byte("bca")) {
		t.Error("back and forth error!")
	}
}

func TestMem(t *testing.T) {
	buf := New(1024)
	checkCap(t, buf, 1024)
	runPerfectSeries(t, buf)
	buf.Reset()
	if n, err := buf.WriteAt([]byte("hello"), 1); err == nil || n != 0 {
		t.Errorf("write should have failed")
	}
}

func TestFilePartition(t *testing.T) {
	buf := NewPartition(NewFilePool(1024, ""))
	checkCap(t, buf, math.MaxInt64)
	runPerfectSeries(t, buf)
	buf.Reset()
}

func TestSmallMulti(t *testing.T) {
	if empty := NewMulti(); empty != nil {
		t.Errorf("the empty buffer should return nil")
	}

	one := NewMulti(New(10))
	if one.Len() != 0 {
		t.Errorf("singleton multi doesn't match inner buffer len")
	}
	if one.Cap() != 10 {
		t.Errorf("singleton multi doesn't match inner buffer cap")
	}
}

func TestMulti(t *testing.T) {
	file, err := ioutil.TempFile("", "buffer")
	if err != nil {
		t.Error(err.Error())
	}
	defer os.Remove(file.Name())
	defer file.Close()

	buf := NewMulti(New(5), New(5), NewFile(500, file), NewPartition(NewFilePool(1024, "")))
	checkCap(t, buf, math.MaxInt64)
	runPerfectSeries(t, buf)
	isPerfectMatch(t, buf, 1024*1024)
	buf.Reset()
}

func TestMulti2(t *testing.T) {
	file, err := ioutil.TempFile("", "buffer")
	if err != nil {
		t.Error(err.Error())
	}
	defer os.Remove(file.Name())
	defer file.Close()

	buf := NewMulti(New(5), New(5), NewFile(500, file))
	checkCap(t, buf, 510)
	runPerfectSeries(t, buf)
	isPerfectMatch(t, buf, 1024*1024)
	buf.Reset()
}

func runPerfectSeries(t *testing.T, buf Buffer) {
	checkEmpty(t, buf)
	simple(t, buf)

	max := int64(1024)
	isPerfectMatch(t, buf, 0)
	for i := int64(1); i < max; i *= 2 {
		isPerfectMatch(t, buf, i)
	}
	isPerfectMatch(t, buf, max)
}

func simple(t *testing.T, buf Buffer) {
	buf.Write([]byte("hello world"))
	data, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal([]byte("hello world"), data) {
		t.Error("Hello world failed.")
	}

	buf.Write([]byte("hello world"))
	data = make([]byte, 3)
	buf.Read(data)
	buf.Write([]byte(" yolo"))
	data, err = ioutil.ReadAll(buf)
	if err != nil {
		t.Error(err.Error())
	}
	if !bytes.Equal([]byte("lo world yolo"), data) {
		t.Error("Buffer crossing error :(", string(data))
	}
}

func buildOutputs(t *testing.T, buf Buffer, size int64) (wrote []byte, read []byte) {
	r := io.LimitReader(rand.Reader, size)
	tee := io.TeeReader(r, buf)

	wrote, _ = ioutil.ReadAll(tee)
	read, _ = ioutil.ReadAll(buf)

	return wrote, read
}

func isPerfectMatch(t *testing.T, buf Buffer, size int64) {
	wrote, read := buildOutputs(t, buf, size)
	if !bytes.Equal(wrote, read) {
		t.Error("Buffer should have matched")
	}
}

func checkEmpty(t *testing.T, buf Buffer) {
	if !Empty(buf) {
		t.Error("Buffer should start empty!")
	}
}

func checkCap(t *testing.T, buf Buffer, correctCap int64) {
	if buf.Cap() != correctCap {
		t.Error("Buffer cap is incorrect", buf.Cap(), correctCap)
	}
}

type bigBuffer struct{}

func (b bigBuffer) Len() int64 { return math.MaxInt64 }

func (b bigBuffer) Cap() int64 { return math.MaxInt64 }

func (b bigBuffer) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func (b bigBuffer) Write(p []byte) (int, error) { return 0, io.ErrShortBuffer }

func (b bigBuffer) Reset() {}

func TestBigList(t *testing.T) {
	l := List([]Buffer{bigBuffer{}, bigBuffer{}})
	if l.Len() != math.MaxInt64 {
		t.Errorf("expected list to be max-size: %d", l.Len())
	}
}

func TestBigMulti(t *testing.T) {
	single := NewMulti(bigBuffer{})
	if single.Len() != math.MaxInt64 {
		t.Errorf("expected len to be max-size: %d", single.Len())
	}
	if single.Cap() != math.MaxInt64 {
		t.Errorf("expected cap to be max-size: %d", single.Cap())
	}

	buf := NewMulti(bigBuffer{}, bigBuffer{})
	if buf.Len() != math.MaxInt64 {
		t.Errorf("expected len to be max-size: %d", buf.Len())
	}
	if buf.Cap() != math.MaxInt64 {
		t.Errorf("expected cap to be max-size: %d", buf.Cap())
	}

	// test defrag failure
	NewMulti(badBuffer{}, badBuffer{})
}

type badBuffer struct{}

func (b badBuffer) Len() int64 { return 1024 }

func (b badBuffer) Cap() int64 { return 10 * 1024 }

func (b badBuffer) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func (b badBuffer) Write(p []byte) (int, error) { return 0, io.ErrShortBuffer }

func (b badBuffer) Reset() {}

func TestBadPartition(t *testing.T) {
	p := NewPool(func() Buffer { return badBuffer{} })
	buf := NewPartition(p)
	_, err := buf.Write([]byte("bad write"))
	if err != io.ErrShortBuffer {
		t.Errorf("wrong read error was returned! %v", err)
	}

	b := make([]byte, 1024)
	_, err = buf.Read(b)
	if err != io.ErrUnexpectedEOF {
		t.Errorf("wrong write error was returned! %v", err)
	}
}

type fakeFile struct {
	name string
}

func (f *fakeFile) Name() string {
	return f.name
}

func (f *fakeFile) Stat() (fi os.FileInfo, err error) {
	return nil, nil
}

func (f *fakeFile) ReadAt(p []byte, off int64) (int, error) {
	return 0, io.EOF
}

func (f *fakeFile) WriteAt(p []byte, off int64) (int, error) {
	return 0, io.ErrShortWrite
}

func (f *fakeFile) Close() error { return nil }

func TestBadGobFile(t *testing.T) {
	b := NewFile(10, &fakeFile{name: "test"})
	b2 := NewFile(10, &fakeFile{name: "test2"})
	b3 := NewMulti(b, b2)
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&b3); err != nil {
		t.Error(err)
	}
	var buffer Buffer
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&buffer); err == nil {
		t.Error("expeced an error here, file does not exist")
	}
}

func TestBadGobFile2(t *testing.T) {
	b := New(10)
	b2 := NewFile(10, &fakeFile{name: "test2"})
	b3 := NewMulti(b, b2)
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(&b3); err != nil {
		t.Error(err)
	}
	var buffer Buffer
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&buffer); err == nil {
		t.Error("expeced an error here, file does not exist")
	}
}
