# cozyish

![](cozyish.png)


### Running
```bash
docker-compose build

docker-compose up
```

### Concept
A crawler searches a provided site url for all png & jpg images, and index's them against the api.  The index payload is 
```json
{
    "image": "http://example.com/some/image/path.jpg"
}
```
The images are then cached, stored, transformed and classified.  A resultant object can be queried, and the final image downloadable. e.g.
```json
{
        "id": "sbykXske",
        "image": "http://example.com/some/image/path.jpg",
        "nsfw_score": 0.0003904797194991261,
        "tags": [
            "digital",
            "wall",
            "analog"
        ]
    }
```

### API
* `:8000/api/index  `     - indexes given the above payload, returns the payload + a generated id field.
* `:8000/api/image  `     - json array of indexed/stored/transformed images
* `:8000/api/image/{id}`  - raw transformed image
* `:4444/{site} `         - without scheme e.g. secretagentsnowman.com


### Dependencies
* Elasticsearch
* Minio
* Docker (docker-compose)
* RabbitMQ
* Yahoo nsfw api
* Deep Detect
* go-exif-remove
* steganography


