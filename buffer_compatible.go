package buffer

import "io"

// Buffer is used to Write() data which will be Read() later.
type BufferCompatible interface {
	Len() int     // How much data is Buffered in bytes
	Cap() int     // How much data can be Buffered at once in bytes.
	Len64() int64 // How much data is Buffered in bytes
	Cap64() int64 // How much data can be Buffered at once in bytes.
	io.Reader     // Read() will read from the top of the buffer [io.EOF if empty]
	io.Writer     // Write() will write to the end of the buffer [io.ErrShortWrite if not enough space]
	Reset()       // Truncates the buffer, Len() == 0.
}

func MakeBufferCompatibe(b Buffer) BufferCompatible {
	return compatible{
		Buffer: b,
	}
}

type compatible struct {
	Buffer
}

func (c compatible) Len() int {
	return int(c.Buffer.Len())
}

func (c compatible) Cap() int {
	return int(c.Buffer.Cap())
}

func (c compatible) Len64() int64 {
	return c.Buffer.Len()
}

func (c compatible) Cap64() int64 {
	return c.Buffer.Cap()
}
