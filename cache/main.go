package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexellis/faas/gateway/metrics"
	elasticsearch "github.com/elastic/go-elasticsearch/v6"
	esapi "github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/gorilla/mux"
	"github.com/streadway/amqp"
)

var RABBITMQ = "localhost:5672"

func main() {

	fmt.Println("Starting cache...")

	if os.Getenv("RABBITMQ") != "" {
		RABBITMQ = os.Getenv("RABBITMQ")
	}

	conn, err := amqp.Dial("amqp://user:bitnami@" + RABBITMQ + "/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()
	q, err := ch.QueueDeclare(
		"cache", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	fmt.Println("queue initialized, consuming...")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	go func() {
		r := mux.NewRouter()
		metricsHandler := metrics.PrometheusHandler()
		r.Handle("/metrics", metricsHandler)
		srv := &http.Server{
			Handler:      r,
			Addr:         "0.0.0.0:8004",
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		}
		srv.ListenAndServe()
	}()

	wait := make(chan bool)
	go func() {
		for d := range msgs {
			var reqBody map[string]interface{}
			err := json.Unmarshal(d.Body, &reqBody)
			if err != nil {
				fmt.Println("error in unmarshalling json " + err.Error())
			} else {
				err = index(reqBody)
				b, _ := json.Marshal(reqBody)
				fmt.Println(string(b))
				if err != nil {
					fmt.Println("error in cacheing data " + err.Error())
				} else {
					q2, err := ch.QueueDeclare(
						"next", // name
						false,  // durable
						false,  // delete when unused
						false,  // exclusive
						false,  // no-wait
						nil,    // arguments
					)
					failOnError(err, "Failed to declare a queue")
					err = ch.Publish(
						"",      // exchange
						q2.Name, // routing key
						false,   // mandatory
						false,   // immediate
						amqp.Publishing{
							ContentType: "application/json",
							Body:        b,
						})
					failOnError(err, "Failed to publish a message")
				}
			}

		}
	}()
	<-wait
}

func failOnError(err error, msg string) {
	if err != nil {
		fmt.Println("%s: %s", msg, err)
	}
}

func index(reqBody map[string]interface{}) error {

	var es, _ = elasticsearch.NewDefaultClient()

	jsonString, _ := json.Marshal(reqBody)

	req := esapi.IndexRequest{
		Index:      "cozyish-images",
		DocumentID: fmt.Sprintf("%f", reqBody["id"].(string)),
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
		return err
	}

	return nil
}
