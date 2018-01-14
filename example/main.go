package main

import (
	"log"
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

		// track sub things
		trackOneThing(r)
		trackTwoThings(r)

		w.Write([]byte("OK"))

	})).ServeHTTP)

	log.Println(http.ListenAndServe(":8000", nil))

}

func trackOneThing(r *http.Request) {
	track, _ := track.New("trackOneThing", r.Context())
	defer track.Close()

	time.Sleep(time.Millisecond * time.Duration(rand.Intn(5)+1))

	track.Close()
}

func trackTwoThings(r *http.Request) {
	ctx := r.Context()

	trackA, ctx := track.New("trackTwoThings", ctx)
	defer trackA.Close()

	{
		trackA1, _ := track.New("trackTwoThings1", ctx)
		defer trackA1.Close()

		time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)+5))

		trackA1.Close()
	}

	{
		trackA2, _ := track.New("trackTwoThings2", ctx)
		defer trackA2.Close()

		time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)+5))

		trackA2.Close()
	}

	trackA.Close()
}
