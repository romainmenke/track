package track_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/romainmenke/track"
)

func TestTrack(t *testing.T) {
	tracker := track.NewTracker()
	go tracker.Start()

	time.Sleep(time.Millisecond * 5)

	defer tracker.Stop()

	f := func(ctx context.Context) {
		track, ctx := track.New("child-op", ctx)
		defer track.Close()

		time.Sleep(time.Microsecond * 10)
	}

	h := tracker.Handler("test-handler", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		track, ctx := track.New("some-op", r.Context())
		defer track.Close()

		track.ErrS("fire!!!!")

		f(ctx)
	}))

	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	r.Header.Set("X-Request-ID", uuid.New())

	for i := 0; i < 10; i++ {
		go h.ServeHTTP(httptest.NewRecorder(), r)
	}

	time.Sleep(time.Millisecond * 5)

}
