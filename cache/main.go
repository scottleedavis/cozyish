package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	elasticsearch "github.com/elastic/go-elasticsearch/v6"
	esapi "github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/streadway/amqp"
)

func main() {

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

	wait := make(chan bool)
	go func() {
		for d := range msgs {
			fmt.Println("Received a message: %s", string(d.Body))
			var reqBody map[string]interface{}
			err := json.Unmarshal(d.Body, &reqBody)
			if err != nil {
				fmt.Println("error in unmarshalling json " + err.Error())
			} else {
				fmt.Println("***** " + reqBody["image"].(string))
				err = index(reqBody)
				if err != nil {
					fmt.Println("error in cacheing data " + err.Error())
				} else {
					q2, err := ch.QueueDeclare(
						"incoming-store", // name
						false,            // durable
						false,            // delete when unused
						false,            // exclusive
						false,            // no-wait
						nil,              // arguments
					)
					failOnError(err, "Failed to declare a queue")
					err = ch.Publish(
						"",      // exchange
						q2.Name, // routing key
						false,   // mandatory
						false,   // immediate
						amqp.Publishing{
							ContentType: "application/json",
							Body:        []byte(d.Body),
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
		return errors.New("Error parse es response")
	}

	fmt.Println("[%s] %s; version=%d", res.Status(), r2["result"], int(r2["_version"].(float64)))

	return nil
}
