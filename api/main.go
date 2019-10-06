package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/streadway/amqp"
)

var cookieStore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

func main() {

	cookieStore.Options = &sessions.Options{
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
	srv.ListenAndServe()
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, "cozyish-store")
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

	queue(reqBody)

	w.Header().Set("Content-Type", "application/json")
	// fmt.Fprintf(w, string(resp))
	fmt.Fprintf(w, "{}")
}

func queue(reqBody map[string]interface{}) {
	// conn, err := amqp.Dial("amqp://user:bitnami@localhost:5672/")
	conn, err := amqp.Dial("amqp://user:bitnami@rabbitmq:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()
	q, err := ch.QueueDeclare(
		"incoming-cache", // name
		false,            // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	failOnError(err, "Failed to declare a queue")

	body, err := json.Marshal(reqBody)
	if err != nil {
		panic(err)
	}
	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
		})
	failOnError(err, "Failed to publish a message")
}

func failOnError(err error, msg string) {
	if err != nil {
		fmt.Println("%s: %s", msg, err)
	}
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
