package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/minio/minio-go/v6"
	exifremove "github.com/scottleedavis/go-exif-remove"
	"github.com/streadway/amqp"
	steganography "gopkg.in/auyer/steganography.v2"
)

var RABBITMQ = "localhost:5672"
var MINIO = "localhost:9000"
var MINIO_ACCESS_KEY = "minioaccesskey"
var MINIO_SECRET_KEY = "miniosecretkey"

func main() {

	if os.Getenv("RABBITMQ") != "" {
		RABBITMQ = os.Getenv("RABBITMQ")
	}

	if os.Getenv("MINIO") != "" {
		MINIO = os.Getenv("MINIO")
	}

	if os.Getenv("MINIO_ACCESS_KEY") != "" {
		MINIO_ACCESS_KEY = os.Getenv("MINIO_ACCESS_KEY")
	}

	if os.Getenv("MINIO_SECRET_KEY") != "" {
		MINIO_SECRET_KEY = os.Getenv("MINIO_SECRET_KEY")
	}

	conn, err := amqp.Dial("amqp://user:bitnami@" + RABBITMQ + "/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()
	q, err := ch.QueueDeclare(
		"incoming-transform", // name
		false,                // durable
		false,                // delete when unused
		false,                // exclusive
		false,                // no-wait
		nil,                  // arguments
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
			var reqBody map[string]interface{}
			err := json.Unmarshal(d.Body, &reqBody)
			if err != nil {
				fmt.Println("error in unmarshalling json " + err.Error())
			} else {
				err = transform(reqBody)
				if err != nil {
					fmt.Println("error in storing data " + err.Error())
				} else {
					q2, err := ch.QueueDeclare(
						"incoming-classify", // name
						false,               // durable
						false,               // delete when unused
						false,               // exclusive
						false,               // no-wait
						nil,                 // arguments
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

func transform(reqBody map[string]interface{}) error {

	p := strings.Split(reqBody["image"].(string), "/")
	filePath := p[len(p)-1]

	endpoint := MINIO
	accessKeyID := MINIO_ACCESS_KEY
	secretAccessKey := MINIO_SECRET_KEY
	useSSL := false

	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		return err
	}

	bucketName := "cozyish-images"
	location := "none"

	found, err := minioClient.BucketExists(bucketName)
	if err != nil {
		fmt.Println("Bucket exists error " + err.Error())
		err = minioClient.MakeBucket(bucketName, location)
		if err != nil {
			fmt.Println("Make bucket error " + err.Error())
			return err
		}
	} else if !found {
		err = minioClient.MakeBucket(bucketName, location)
		if err != nil {
			fmt.Println("Make bucket error " + err.Error())
			return err
		}
	}

	err = minioClient.FGetObject(bucketName, reqBody["id"].(string), filePath, minio.GetObjectOptions{})
	if err != nil {
		fmt.Println(err)
		return err
	}

	if data, err := ioutil.ReadFile(filePath); err != nil {
		fmt.Printf(err.Error())
		return err
	} else {
		if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
			fmt.Printf("ERROR: original image is corrupt" + err.Error() + "\n")
			return err
		}
		//strip exif
		if filtered, err := exifremove.Remove(data); err != nil {
			return err
		} else {
			if _, _, err = image.Decode(bytes.NewReader(filtered)); err != nil {
				fmt.Printf("ERROR: filtered image is corrupt" + err.Error() + "\n")
				return err
			}

			//remove stegonagraphy
			img, _, err := image.Decode(bytes.NewReader(filtered))
			if err != nil {
				return err
			}

			w := &bytes.Buffer{}
			err = steganography.Encode(w, img, []byte(""))
			ioutil.WriteFile(filePath, filtered, 0644)
		}
	}

	_, err = minioClient.FPutObject(bucketName, reqBody["id"].(string), filePath, minio.PutObjectOptions{ContentType: "application/png"})
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

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
