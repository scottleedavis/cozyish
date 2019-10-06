package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/minio/minio-go/v6"
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
		"incoming-store", // name
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
				err = store(reqBody)
				if err != nil {
					fmt.Println("error in storing data " + err.Error())
				} else {
					q2, err := ch.QueueDeclare(
						"incoming-transform", // name
						false,                // durable
						false,                // delete when unused
						false,                // exclusive
						false,                // no-wait
						nil,                  // arguments
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

func store(reqBody map[string]interface{}) error {

	p := strings.Split(reqBody["image"].(string), "/")
	filePath := p[len(p)-1]

	if err := DownloadFile(filePath, reqBody["image"].(string)); err != nil {
		panic(err)
	}

	endpoint := "minio:9000"
	accessKeyID := "minioaccesskey"
	secretAccessKey := "miniosecretkey"
	useSSL := false

	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		fmt.Println(err)
	}

	bucketName := "cozyish-file-store"
	location := "none"

	found, err := minioClient.BucketExists(bucketName)
	if err != nil {
		fmt.Println("Bucket exists error " + err.Error())
		err = minioClient.MakeBucket(bucketName, location)
		if err != nil {
			fmt.Println("Make bucket error " + err.Error())
			return err
		}
	}
	if found {
		fmt.Println("Bucket found")
	}

	n, err := minioClient.FPutObject(bucketName, fmt.Sprintf("%f", reqBody["id"].(float64)), filePath, minio.PutObjectOptions{ContentType: "application/png"})
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	fmt.Println("Successfully uploaded of size %d\n", n)
	return nil
}

func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
