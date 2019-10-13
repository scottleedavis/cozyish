package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	elasticsearch "github.com/elastic/go-elasticsearch/v6"
	esapi "github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/streadway/amqp"
)

var RABBITMQ = "localhost:5672"
var MINIO = "localhost:9000"
var MINIO_ACCESS_KEY = "minioaccesskey"
var MINIO_SECRET_KEY = "miniosecretkey"
var NSFWAPI = "localhost:5000"
var DEEPDETECT = "localhost:8080"

type NSFW struct {
	Score float64 `json:"score"`
	URL   string  `json:"url"`
}

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

	if os.Getenv("NSFWAPI") != "" {
		NSFWAPI = os.Getenv("NSFWAPI")
	}

	if os.Getenv("DEEPDETECT") != "" {
		DEEPDETECT = os.Getenv("DEEPDETECT")
	}

	data := `{
		"description": "image classification service",
		"mllib": "caffe",
		"model": {
			"init": "https://deepdetect.com/models/init/desktop/images/classification/ilsvrc_googlenet.tar.gz",
			"repository": "/opt/models/ilsvrc_googlenet",
		"create_repository": true
		},
		"parameters": {
			"input": {
				"connector": "image"
			}
		},
		"type": "supervised"
	}`
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, "http://"+DEEPDETECT+"/services/imageserv", bytes.NewReader([]byte(data)))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	_, err = client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	conn, err := amqp.Dial("amqp://user:bitnami@" + RABBITMQ + "/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()
	q, err := ch.QueueDeclare(
		"classify", // name
		false,      // durable
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
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
				fileProperties, err := classify(reqBody)
				b, _ := json.Marshal(fileProperties)
				// fmt.Println(string(b))
				if err != nil {
					fmt.Println("error in classifying data " + err.Error())
				} else {
					q2, err := ch.QueueDeclare(
						"cache", // name
						false,   // durable
						false,   // delete when unused
						false,   // exclusive
						false,   // no-wait
						nil,     // arguments
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

func classify(reqBody map[string]interface{}) (map[string]interface{}, error) {

	fileProperties := reqBody

	nsfw, err := NsfwScore(reqBody)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fileProperties["nsfw_score"] = nsfw.Score

	tags, err := ImageClassify(reqBody)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fileProperties["tags"] = tags

	return fileProperties, nil
}

func ImageClassify(reqBody map[string]interface{}) ([]string, error) {

	buf := []byte(`{
		"service":"imageserv",
		"parameters":{
		  "output":{
			"best":3
		  }
		},
		"data":["` + reqBody["image"].(string) + `"]
	  }`)

	resp, err := http.Post("http://"+DEEPDETECT+"/predict", "application/json", bytes.NewReader(buf))
	if err != nil {
		fmt.Println("Error in predict: " + err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var bb map[string]interface{}
	err = json.Unmarshal(body, &bb)
	if err != nil {
		return nil, err
	}

	tags := []string{}

	if bb["body"] == nil {
		return tags, nil
	}

	predictions := (bb["body"].(map[string]interface{})["predictions"]).([]interface{})
	if len(predictions) == 0 {
		return tags, nil
	}
	classes := (predictions[0]).(map[string]interface{})["classes"].([]interface{})
	for _, cc := range classes {
		c := cc.(map[string]interface{})
		cat := c["cat"].(string)
		cat = strings.Replace(cat, ",", "", -1)
		t := strings.Split(cat, " ")
		if len(t) > 1 {
			t = t[1 : len(t)-1]
			if len(t) > 0 {
				tags = append(tags, t[0])
			}
		}
	}

	return tags, nil
}

func NsfwScore(reqBody map[string]interface{}) (NSFW, error) {
	var nsfw NSFW
	resp, err := http.Get("http://" + NSFWAPI + "/?url=" + reqBody["image"].(string))
	if err != nil {
		fmt.Printf("ERROR: unable to classify nsfw_api" + err.Error() + "\n")
		return nsfw, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nsfw, err
	}

	err = json.Unmarshal(body, &nsfw)
	if err != nil {
		return nsfw, err
	}

	return nsfw, nil
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
		fmt.Println("Error parsing the response body: %s", err)
		return errors.New("Error parse es response")
	}

	return nil
}
