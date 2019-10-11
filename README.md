# cozyish

![](cozyish.png)

### Concept
A crawler searches a provided site url for all png & jpg images, and index's them against the api.  The index payload is 
```json
{
    "image": "http://example.com/some/image/path.jpg"
}
```
The images are then cached, stored, analyzed and classified. A resultant object can be queried, and the final image downloadable. e.g.
```json
{
    "exif": [
        ...
        {
            "GPSLatitudeRef": "N"
        },
        {
            "GPSLatitude": "26/1"
        },
        {
            "GPSLongitudeRef": "W"
        },
        {
            "GPSLongitude": "80/1"
        },
        ...
    ],
    "id": "ähJäYZöh",
    "image": "path.jpg",
    "nsfw_score": 0.016476402059197426,
    "steganography": "This message was hidden in the image.",
    "tags": [
        "child",
        "ball"
    ]
}
```
	
* [NSFW classifier by yahoo](https://github.com/yahoo/open_nsfw): classifier with nudity score 0 to 1
* [Classification tags by Deepdetect](https://www.deepdetect.com):   Currently using [the ilsrvc_googlenet pretrained model](https://www.deepdetect.com/models/ilsvrc_googlenet/).  
* [EXIF reader](https://github.com/dsoprea/go-exif)
* [LSB-Steganography reader](https://github.com/auyer/steganography) 

### Running
```bash
docker-compose build

docker-compose up
```

### Usage
1) Crawl a website for its images using the sitename as a path parameter. e.g. `localhost:4444/?url=https://sitename.com`.   (alternatively, index a single image at `localhost:8000/api/index`)
2) View all indexed, transformed and classified images. e.g. `localhost:8000/api/image`


### API
* `:8000/api/index  `     - indexes given the above payload, returns the payload + a generated id field.
* `:8000/api/image  `     - json array of indexed/stored/transformed images
* `:8000/api/image/{id}`  - raw transformed image
* `:4444/?url={site} `    - site e.g. http://secretagentsnowman.com



### Dependencies
* [Elasticsearch](https://www.elastic.co/)
* [Minio](https://min.io/)
* [Docker](https://www.docker.com/) & [(docker-compose)](https://docs.docker.com/compose/)
* [RabbitMQ](https://www.rabbitmq.com/)
* [Yahoo open nsfw model](https://github.com/yahoo/open_nsfw)
* [Deep Detect](https://www.deepdetect.com/)
* [go-exif](https://github.com/dsoprea/go-exif)
* [steganography](https://github.com/auyer/steganography)


