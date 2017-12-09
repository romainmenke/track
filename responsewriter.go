package track

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

type headerUpdater struct {
	http.ResponseWriter
	code         int
	updateHeader func(int, http.Header) int
}

type stringWriter interface {
	WriteString(s string) (n int, err error)
}

func (w *headerUpdater) WriteHeader(code int) {
	if w.code != 0 {
		return
	}

	if code == 0 {
		code = http.StatusOK
	}

	w.code = code

	w.code = w.updateHeader(w.code, w.Header())

	w.ResponseWriter.WriteHeader(w.code)
}

func (w *headerUpdater) Write(b []byte) (int, error) {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (w *headerUpdater) WriteString(s string) (int, error) {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}

	if sw, ok := w.ResponseWriter.(stringWriter); ok {
		return sw.WriteString(s)
	}

	return w.ResponseWriter.Write([]byte(s))
}

func (w *headerUpdater) CloseNotify() <-chan bool {
	return w.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

func (w *headerUpdater) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

type h1 struct{ *headerUpdater }

func (w *h1) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func (w *h1) ReadFrom(reader io.Reader) (int64, error) {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.(io.ReaderFrom).ReadFrom(reader)
}

type h2 struct{ *headerUpdater }

func (w *h2) Push(target string, opts *http.PushOptions) error {
	return w.ResponseWriter.(http.Pusher).Push(target, opts)
}
