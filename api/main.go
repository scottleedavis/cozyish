package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v6"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/minio/minio-go/v6"
	"github.com/streadway/amqp"
)

const (
	RABBITMQ = "rabbitmq:5672"
	MINIO    = "minio:9000"
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
	r.HandleFunc("/api/image", ImageListHandler)
	r.HandleFunc("/api/image/{id}", ImageHandler)

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
	fmt.Fprintf(w, "{}")
}

func ImageListHandler(w http.ResponseWriter, r *http.Request) {
	found, err := search()
	if err != nil {
		fmt.Println("Error searching: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	j, err := json.Marshal(found)
	w.Write(j)
}

func ImageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileProperties, err := get(vars["id"])
	if err != nil {
		fmt.Println("Error finding image: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	bytes, err := download(fileProperties)
	if err != nil {
		fmt.Println("Error download image: %s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch filepath.Ext(fileProperties["image"].(string)) {
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg":
		w.Header().Set("Content-Type", "image/jpg")
	case ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	}
	w.Write(bytes)
}

func download(fileProperties map[string]interface{}) ([]byte, error) {
	endpoint := MINIO
	accessKeyID := "minioaccesskey"
	secretAccessKey := "miniosecretkey"
	useSSL := false

	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	bucketName := "cozyish-file-store"
	location := "none"

	found, err := minioClient.BucketExists(bucketName)
	if err != nil {
		fmt.Println("Bucket exists error " + err.Error())
		err = minioClient.MakeBucket(bucketName, location)
		if err != nil {
			fmt.Println("Make bucket error " + err.Error())
			return nil, err
		}
	}
	if found {
		fmt.Println("Bucket found")
	}

	p := strings.Split(fileProperties["image"].(string), "/")
	filePath := p[len(p)-1]
	err = minioClient.FGetObject(bucketName, fmt.Sprintf("%f", fileProperties["id"].(float64)), filePath, minio.GetObjectOptions{})
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading image file " + err.Error())
		return nil, err
	}

	return data, nil

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
		flt := (hit.(map[string]interface{})["_id"]).(string)
		s, _ := strconv.ParseFloat(flt, 64)
		iflt := int(s)
		if strconv.Itoa(iflt) == id {
			doc := hit.(map[string]interface{})["_source"]
			fileProperties = doc.(map[string]interface{})
			break
		}

	}
	return fileProperties, nil
}

func search() ([]map[string]interface{}, error) {

	var es, _ = elasticsearch.NewDefaultClient()

	var (
		r map[string]interface{}
	)
	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
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

	var found []map[string]interface{}
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		found = append(found, hit.(map[string]interface{})["_source"].(map[string]interface{}))
	}
	return found, nil
}

func queue(reqBody map[string]interface{}) {
	// conn, err := amqp.Dial("amqp://user:bitnami@localhost:5672/")
	conn, err := amqp.Dial("amqp://user:bitnami@" + RABBITMQ + "/")
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