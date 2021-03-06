package track

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-logfmt/logfmt"
	"github.com/pborman/uuid"
)

type Logger interface {
	Print(...interface{})
}

type defaultLogger struct{}

func (l *defaultLogger) Print(v ...interface{}) {
	fmt.Print(v...)
}

type opKeyType string

const opKey = opKeyType("track/op")

var DefaultLogger Logger = &defaultLogger{}

func New(name string, ctx context.Context) (Op, context.Context) {
	if o, ok := ctx.Value(opKey).(*op); ok && o != nil {
		c := o.child(name)
		ctx = contextWithOp(ctx, c)
		return c, ctx
	}

	return nil, ctx
}

func contextWithOp(ctx context.Context, val Op) context.Context {
	return context.WithValue(ctx, opKey, val)
}

func TrackIDFromContext(ctx context.Context) string {
	if o, ok := ctx.Value(opKey).(*op); ok && o != nil {
		o.Lock()
		defer o.Unlock()
		return o.trackID
	}

	return uuid.New()
}

type Op interface {
	Meta(string, string)
	Err(error) error
	ErrS(string) error
	Close()
	Name() string
}

type op struct {
	sync.Mutex

	children []*op
	close    closeFunc
	closed   bool
	duration time.Duration
	err      string
	meta     map[string]string
	name     string
	request  map[string]string
	start    time.Time
	trackID  string
}

func newOp(name string, trackID string) *op {
	o := &op{
		meta:    make(map[string]string),
		name:    name,
		request: make(map[string]string),
		start:   time.Now(),
		trackID: trackID,
	}

	return o
}

func (o *op) child(name string) *op {
	c := newOp(name, o.trackID)

	o.Lock()
	defer o.Unlock()

	o.children = append(o.children, c)

	c.close = closeFunc(func() {
		c.Lock()
		defer c.Unlock()
		if c.closed == true {
			return
		}

		c.duration = time.Since(c.start)
		c.closed = true
	})

	return c
}

type closeFunc func()

func (f closeFunc) Close() {
	f()
}

func (o *op) encodeServerTime() string {
	return strings.Join(o.collectServerTime(nil), ", ")
}

func (o *op) collectServerTime(buf []string) []string {
	o.Lock()
	defer o.Unlock()

	var out []string
	if buf == nil {
		out = make([]string, 0, len(o.children)+2)
	} else {
		out = buf
	}

	if o.closed {
		out = append(out, fmt.Sprintf("%s;dur=%f;desc=\"%s\";", o.name, o.duration.Seconds()*1000, o.name))
	} else {
		out = append(out, fmt.Sprintf("%s;dur=%f;desc=\"%s\";", o.name, time.Since(o.start).Seconds()*1000, o.name))
	}

	for _, c := range o.children {
		out = c.collectServerTime(out)
	}

	return out
}

func (o *op) print() {
	o.Lock()
	defer o.Unlock()

	buf := bytes.NewBuffer(nil)
	e := logfmt.NewEncoder(buf)
	defer func() {
		DefaultLogger.Print(string(buf.Bytes()))
	}()

	err := e.EncodeKeyval("op", o.name)
	if err != nil {
		panic(err)
	}

	if len(o.request) > 0 {
		for k, v := range o.request {
			err = e.EncodeKeyval(k, v)
			if err != nil {
				panic(err)
			}
		}
	}

	if len(o.meta) > 0 {
		for k, v := range o.meta {
			err = e.EncodeKeyval(k, v)
			if err != nil {
				panic(err)
			}
		}
	}

	if o.err != "" {
		err = e.EncodeKeyval("err", o.err)
		if err != nil {
			panic(err)
		}
	}

	err = e.EncodeKeyval("t", o.start)
	if err != nil {
		panic(err)
	}
	err = e.EncodeKeyval("dur", o.duration)
	if err != nil {
		panic(err)
	}
	err = e.EncodeKeyval("track", o.trackID)
	if err != nil {
		panic(err)
	}

	err = e.EndRecord()
	if err != nil {
		panic(err)
	}

	for _, c := range o.children {
		c.print()
	}
}

func (o *op) Close() {
	o.close()
}

func (o *op) Err(err error) error {
	o.Lock()
	defer o.Unlock()

	o.err = err.Error()
	return err
}

func (o *op) ErrS(err string) error {
	return o.Err(errors.New(err))
}

func (o *op) Meta(key string, value string) {
	o.Lock()
	defer o.Unlock()

	o.meta[key] = value
}

func (o *op) Name() string {
	return o.name
}

func (o *op) req(r *http.Request) {
	o.Lock()
	defer o.Unlock()

	o.request["method"] = r.Method

	if r.URL != nil {
		o.request["path"] = r.URL.String()
	}
}
