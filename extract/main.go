package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/dsoprea/go-exif"
	elasticsearch "github.com/elastic/go-elasticsearch/v6"
	esapi "github.com/elastic/go-elasticsearch/v6/esapi"
	"github.com/minio/minio-go/v6"
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
		"incoming-extract", // name
		false,              // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
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
				err = extract(reqBody)
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

type IfdEntry struct {
	IfdPath     string      `json:"ifd_path"`
	FqIfdPath   string      `json:"fq_ifd_path"`
	IfdIndex    int         `json:"ifd_index"`
	TagId       uint16      `json:"tag_id"`
	TagName     string      `json:"tag_name"`
	TagTypeId   uint16      `json:"tag_type_id"`
	TagTypeName string      `json:"tag_type_name"`
	UnitCount   uint32      `json:"unit_count"`
	Value       interface{} `json:"value"`
	ValueString string      `json:"value_string"`
}

func extract(reqBody map[string]interface{}) error {

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

		var fileProperties map[string]interface{}

		steganography_msg := ""
		var img image.Image
		reader := bytes.NewReader(data)    // buffer reader
		img, _, err = image.Decode(reader) // decoding to golang's image.Image
		if err == nil {
			sizeOfMessage := steganography.GetMessageSizeFromImage(img) // retrieving message size to decode in the next line

			steganography_msg = string(steganography.Decode(sizeOfMessage, img)) // decoding the message from the file
			fmt.Println("steganography: " + steganography_msg)
		} else {
			fmt.Println("unable to decode file")
		}

		exifEntries := []map[string]string{}
		if rawExif, err := exif.SearchAndExtractExif(data); err != nil {
			return nil
		} else {

			im := exif.NewIfdMappingWithStandard()
			ti := exif.NewTagIndex()

			entries := make([]IfdEntry, 0)
			visitor := func(fqIfdPath string, ifdIndex int, tagId uint16, tagType exif.TagType, valueContext exif.ValueContext) (err error) {
				defer func() {
					if state := recover(); state != nil {
					}
				}()

				ifdPath, err := im.StripPathPhraseIndices(fqIfdPath)
				if err != nil {
					return err
				}

				it, err := ti.Get(ifdPath, tagId)
				if err != nil {
					return err
				}

				valueString := ""
				var value interface{}
				if tagType.Type() == exif.TypeUndefined {
					var err2 error
					value, err2 = exif.UndefinedValue(ifdPath, tagId, valueContext, tagType.ByteOrder())
					if err2 != nil {
						return err2
					} else {
						valueString = fmt.Sprintf("%v", value)
					}
				} else {
					valueString, err = tagType.ResolveAsString(valueContext, true)
					if err != nil {
						return err
					}

					value = valueString
				}

				entry := IfdEntry{
					IfdPath:     ifdPath,
					FqIfdPath:   fqIfdPath,
					IfdIndex:    ifdIndex,
					TagId:       tagId,
					TagName:     it.Name,
					TagTypeId:   tagType.Type(),
					TagTypeName: tagType.Name(),
					UnitCount:   valueContext.UnitCount,
					Value:       value,
					ValueString: valueString,
				}

				entries = append(entries, entry)

				return nil
			}

			_, err = exif.Visit(exif.IfdStandard, im, ti, rawExif, visitor)
			if err == nil {
				fmt.Println("exif: " + fmt.Sprintf("%v", entries))
				for _, e := range entries {
					fmt.Println(e.TagName + " " + e.ValueString)
					x := map[string]string{}
					x[e.TagName] = e.ValueString
					exifEntries = append(exifEntries, x)
				}
			}

			fileProperties, err = get(reqBody["id"].(string))
			if err != nil {
				return err
			}

		}

		fileProperties["exif"] = exifEntries
		fileProperties["steganography"] = steganography_msg

		index(fileProperties)

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

	return nil
}
