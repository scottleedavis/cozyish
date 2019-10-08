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
var NSFWAPI = "localhost:5000"

func main() {

	if os.Getenv("RABBITMQ") != "" {
		RABBITMQ = os.Getenv("RABBITMQ")
	}

	if os.Getenv("MINIO") != "" {
		MINIO = os.Getenv("MINIO")
	}

	if os.Getenv("NSFWAPI") != "" {
		NSFWAPI = os.Getenv("NSFWAPI")
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
			fmt.Println("Received a message: %s", string(d.Body))
			var reqBody map[string]interface{}
			err := json.Unmarshal(d.Body, &reqBody)
			if err != nil {
				fmt.Println("error in unmarshalling json " + err.Error())
			} else {
				fmt.Println("***** " + reqBody["image"].(string))
				err = classify(reqBody)
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
	accessKeyID := "minioaccesskey"
	secretAccessKey := "miniosecretkey"
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
	} else if found {
		fmt.Println("Bucket found")
	} else {
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

	resp, err := http.Get("http://" + NSFWAPI + "/?url=" + reqBody["image"].(string))
	if err != nil {
		fmt.Printf("ERROR: unable to classify nswf_api" + err.Error() + "\n")
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error in read resp body " + err.Error())
		return err
	}

	type NSWF struct {
		Score float64 `json:"score"`
		URL   string  `json:"url"`
	}
	var nswf NSWF
	err = json.Unmarshal(body, &nswf)
	if err != nil {
		fmt.Println("ErroORR " + err.Error())
		return err
	}

	fmt.Println("nswf classifier" + fmt.Sprintf("%f", nswf.Score) + " " + nswf.URL)
	fileProperties, err := get(reqBody["id"].(string))
	if err != nil {
		fmt.Println("Error finding image: %s", err)
		return err
	}

	//update cache with classification
	fileProperties["nsfw_score"] = nswf.Score
	index(fileProperties)

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

func get(id string) (map[string]interface{}, error) {

	var es, _ = elasticsearch.NewDefaultClient()

	var (
		r map[string]interface{}
	)
	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"id": id,
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
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		check_id := (hit.(map[string]interface{})["_id"]).(string)
		check_id = fmt.Sprintf("%v", check_id)
		check_id = strings.TrimPrefix(check_id, "%!f(string=")
		check_id = strings.TrimSuffix(check_id, ")")
		if check_id == id {
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

	fmt.Println("[%s] %s; version=%d", res.Status(), r2["result"], int(r2["_version"].(float64)))

	return nil
}
