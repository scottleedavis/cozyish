package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v6"
	"github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
var es, _ = elasticsearch.NewDefaultClient()

func main() {

	store.Options = &sessions.Options{
		MaxAge:   1,
		Secure:   false,
		HttpOnly: false,
	}

	r := mux.NewRouter()
	r.HandleFunc("/api/index", IndexHandler)
	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	fmt.Println(elasticsearch.Version)
	fmt.Println(es.Info())
	srv.ListenAndServe()
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "cozyish-store")
	if session.Values["client"] == nil {
		session.Values["client"] = randomId()
	}
	session.Save(r, w)

	var reqBody map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		fmt.Println("Error parsing the response body: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := index(reqBody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(resp))

}

func index(reqBody map[string]interface{}) ([]byte, error) {

	jsonString, _ := json.Marshal(reqBody)

	req := esapi.IndexRequest{
		Index:      "cozyish-test",
		DocumentID: fmt.Sprintf("%f", reqBody["id"].(float64)),
		Body:       strings.NewReader(string(jsonString)),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), es)
	if err != nil {
		fmt.Println("Error getting response: %s", err)
	}
	defer res.Body.Close()

	var r2 map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&r2)
	if err != nil {
		fmt.Println("Error parsing the response body: %s", err)
		return nil, errors.New("Error parse es response")
	}

	fmt.Println("[%s] %s; version=%d", res.Status(), r2["result"], int(r2["_version"].(float64)))

	return jsonString, nil
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
