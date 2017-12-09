package track_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/fd/httpmiddlewarevet"
	"github.com/romainmenke/track"
)

func TestMiddleware_Handler(t *testing.T) {
	tracker := track.NewTracker()
	go tracker.Start()

	time.Sleep(time.Millisecond * 5)

	defer tracker.Stop()

	httpmiddlewarevet.Vet(t, func(h http.Handler) http.Handler {
		return tracker.Handler("vet-test", h)
	})
}
