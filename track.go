package track

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pborman/uuid"
)

func NewTracker() *tracker {
	return &tracker{
		quitChan: make(chan struct{}),
		opChan:   make(chan *op, 100),
	}
}

type Tracker interface {
	Start()
	Stop()
	Handler(string, http.Handler) http.Handler
	RoundTripper(name string, next http.RoundTripper) http.RoundTripper
}

type tracker struct {
	quitChan chan struct{}
	opChan   chan *op
}

func (t *tracker) Start() (err error) {
	for {
		select {
		case o := <-t.opChan:
			o.print()
		case <-t.quitChan:
			return
		}
	}
}

func (t *tracker) Stop() {
	close(t.quitChan)
}

func (t *tracker) Handler(name string, next http.Handler) http.Handler {

	headerUpdateFunc := func(ctx context.Context) func(code int, h http.Header) int {
		return func(code int, h http.Header) int {

			if o, ok := ctx.Value(opKey).(*op); ok && o != nil {
				h.Add("Server-Timing", o.encodeServerTime())
			}
			return code
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Trailer", "Server-Timing")
		start := time.Now()

		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New()
		}

		xRequestStart := r.Header.Get("X-Request-Start")

		o := newOp(name, requestID)

		if i, err := strconv.ParseInt(xRequestStart, 0, 64); err == nil && i != 0 {
			c := o.child("dyno")
			o.start = time.Unix(0, i*1000000)
			c.start = time.Unix(0, i*1000000)
			c.Close()
		}

		o.close = closeFunc(func() {
			o.Lock()
			defer o.Unlock()

			if o.closed {
				return
			}

			o.duration = time.Since(o.start)
			o.closed = true

			select {
			case t.opChan <- o:
			default:
			}
		})
		defer o.Close()

		o.req(r)

		ctx := contextWithOp(r.Context(), o)

		if _, ok := w.(http.Pusher); ok {
			w = &h2{
				&headerUpdater{
					w,
					0,
					headerUpdateFunc(ctx),
				},
			}
		} else {
			w = &h1{
				&headerUpdater{
					w,
					0,
					headerUpdateFunc(ctx),
				},
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))

		w.Header().Set("Server-Timing", fmt.Sprintf("total;dur=%f", time.Since(start).Seconds()*1000))
	})
}

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (t *tracker) RoundTripper(name string, next http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {

		o := newOp(name, uuid.New())

		o.close = closeFunc(func() {
			o.Lock()
			defer o.Unlock()

			if o.closed {
				return
			}

			o.duration = time.Since(o.start)
			o.closed = true

			select {
			case t.opChan <- o:
			default:
			}
		})
		defer o.Close()

		o.req(req)

		ctx := contextWithOp(req.Context(), o)
		return next.RoundTrip(req.WithContext(ctx))
	})
}
