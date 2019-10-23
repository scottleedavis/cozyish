package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexellis/faas/gateway/metrics"
	"github.com/dsoprea/go-exif"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v6"
	"github.com/streadway/amqp"
	steganography "gopkg.in/auyer/steganography.v2"
)

var RABBITMQ = "localhost:5672"
var MINIO = "localhost:9000"
var MINIO_ACCESS_KEY = "minioaccesskey"
var MINIO_SECRET_KEY = "miniosecretkey"

func main() {

	fmt.Println("Starting extract...")

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
		"extract", // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
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
			Addr:         "0.0.0.0:8002",
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
				fileProperties, err := extract(reqBody)
				b, _ := json.Marshal(fileProperties)
				// fmt.Println(string(b))
				if err != nil {
					fmt.Println("error in storing data " + err.Error())
				} else {
					q2, err := ch.QueueDeclare(
						"classify", // name
						false,      // durable
						false,      // delete when unused
						false,      // exclusive
						false,      // no-wait
						nil,        // arguments
					)
					failOnError(err, "Failed to declare a queue")
					err = ch.Publish(
						"",      // exchange
						q2.Name, // routing key
						false,   // mandatory
						false,   // immediate
						amqp.Publishing{
							ContentType: "application/json",
							Body:        []byte(b),
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

func extract(reqBody map[string]interface{}) (map[string]interface{}, error) {

	p := strings.Split(reqBody["image"].(string), "/")
	filePath := p[len(p)-1]

	endpoint := MINIO
	accessKeyID := MINIO_ACCESS_KEY
	secretAccessKey := MINIO_SECRET_KEY
	useSSL := false

	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		return nil, err
	}

	bucketName := "cozyish-images"
	location := "none"

	found, err := minioClient.BucketExists(bucketName)
	if err != nil {
		fmt.Println("Bucket exists error " + err.Error())
		err = minioClient.MakeBucket(bucketName, location)
		if err != nil {
			fmt.Println("Make bucket error " + err.Error())
			return nil, err
		}
	} else if !found {
		err = minioClient.MakeBucket(bucketName, location)
		if err != nil {
			fmt.Println("Make bucket error " + err.Error())
			return nil, err
		}
	}

	err = minioClient.FGetObject(bucketName, reqBody["id"].(string), filePath, minio.GetObjectOptions{})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	fileProperties := reqBody

	if data, err := ioutil.ReadFile(filePath); err != nil {
		fmt.Printf(err.Error())
		return nil, err
	} else {

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
			fmt.Println("unable to extract exif from file")
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

		}

		fileProperties["exif"] = exifEntries
		fileProperties["steganography"] = steganography_msg
	}

	return fileProperties, nil
}
