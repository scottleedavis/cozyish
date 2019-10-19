# cozyish

![](cozyish.png)
![](screenshot.png)

### Concept
* The crawler searches a site url for all png & jpg images and index's them against the api.  
* The video node converts video streams to indexed jpg images. 

The images are then stored, analyzed, classified and cached. The resultant objects can be queried, and the final image downloadable. e.g.
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
Two ingestors:

1) Crawl a website for its images using the sitename as a path parameter. 
e.g. 
* `localhost:4444/?url=https://sitename.com`.   
* alternatively, index a single image at `POST localhost:8000/api/index
{
	"image": "http://example.com/path/to/image.png"
}
`

2) Record a video that is converted to images 
* `localhost:3000`

3) View all indexed, transformed and classified images. e.g. `localhost:8000/api/image`


### API
* `:8000/api/index  `     - indexes given the above payload, returns the payload + a generated id field.
* `:8000/api/image  `     - json array of images
* `:8000/api/image/{id}`  - raw image
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


