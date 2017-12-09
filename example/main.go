package main

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/romainmenke/track"
)

func main() {

	tracker := track.NewTracker()
	go tracker.Start()
	defer tracker.Stop()
	time.Sleep(time.Millisecond * 5)

	http.HandleFunc("/", tracker.Handler("main", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		track1, ctx := track.New("abc", r.Context())
		defer track1.Close()

		time.Sleep(time.Millisecond * time.Duration(rand.Intn(5)))
		track1.Close()

		track2, _ := track.New("xyz", ctx)
		defer track2.Close()

		time.Sleep(time.Millisecond * time.Duration(rand.Intn(5)))

		track2b, _ := track.New("rst", ctx)
		defer track2b.Close()
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(5)))
		track2b.Close()

		w.Write([]byte("OK"))

	})).ServeHTTP)

	panic(http.ListenAndServe(":8000", nil))

}
