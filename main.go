package main

//server
//server page with session id

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

func main() {

	store.Options = &sessions.Options{
		MaxAge:   1,
		Secure:   false,
		HttpOnly: false,
	}

	r := mux.NewRouter()
	r.HandleFunc("/", HomeHandler)
	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	srv.ListenAndServe()
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	session, _ := store.Get(r, "cozyish-store")
	if session.Values["client"] == nil {
		session.Values["client"] = randomId()
	}
	session.Save(r, w)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Why Hello there %s\n", session.Values["client"])
}

func randomId() string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZÅÄÖ" +
		"abcdefghijklmnopqrstuvwxyzåäö" +
		"0123456789")
	length := 8
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}
