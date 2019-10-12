package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	elasticsearch "github.com/elastic/go-elasticsearch/v6"
	esapi "github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/minio/minio-go/v6"
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
		"incoming-classify", // name
		false,               // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
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
				err = classify(reqBody)
				b, _ := json.Marshal(reqBody)
				fmt.Println(string(b))
				if err != nil {
					fmt.Println("error in classifying data " + err.Error())
				} else {
					q2, err := ch.QueueDeclare(
						"incoming-next", // name
						false,           // durable
						false,           // delete when unused
						false,           // exclusive
						false,           // no-wait
						nil,             // arguments
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

func classify(reqBody map[string]interface{}) error {

	p := strings.Split(reqBody["image"].(string), "/")
	filePath := p[len(p)-1]

	endpoint := MINIO
	accessKeyID := MINIO_ACCESS_KEY
	secretAccessKey := MINIO_SECRET_KEY
	useSSL := false

	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		fmt.Println(err)
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

	fileProperties, err := get(reqBody)
	if err != nil {
		fmt.Println("Error finding image: %s", err)
		return err
	}

	nsfw, err := NsfwScore(reqBody)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fileProperties["nsfw_score"] = nsfw.Score

	tags, err := ImageClassify(reqBody)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fileProperties["tags"] = tags

	index(fileProperties)

	return nil
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

func get(reqBody map[string]interface{}) (map[string]interface{}, error) {

	var es, _ = elasticsearch.NewDefaultClient()

	var (
		r map[string]interface{}
	)
	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"id": reqBody["id"].(string),
			},
		},
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		fmt.Println("Error encoding query: %s", err)
		return nil, err
	}

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("cozyish-images"),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithPretty(),
	)
	if err != nil {
		fmt.Println("Error getting response: %s", err)
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			fmt.Println("Error parsing the response body: %s", err)
			return nil, err
		} else {
			fmt.Println("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		fmt.Println("Error parsing the response body: %s", err)
		return nil, err
	}

	var fileProperties map[string]interface{}
	fileProperties["id"] = reqBody["id"].(string)
	fileProperties["image"] = reqBody["image"].(string)

	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		check_id := (hit.(map[string]interface{})["_id"]).(string)
		check_id = fmt.Sprintf("%v", check_id)
		check_id = strings.TrimPrefix(check_id, "%!f(string=")
		check_id = strings.TrimSuffix(check_id, ")")
		if check_id == reqBody["id"].(string) {
			doc := hit.(map[string]interface{})["_source"]
			fileProperties = doc.(map[string]interface{})
			break
		}

	}

	return fileProperties, nil
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
