# cozyish

_todo_
* classify images

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
    "image": "http://example.com/some/image/path.jpg",
    "tags": ["optional", "tags"]
}
```
The images are then cached, stored, transformed and classified.

### API
* `:8000/api/index  `     - indexes given the above payload, returns the payload + a generated id field.
* `:8000/api/image  `     - json array of indexed/stored/transformed images
* `:8000/api/image/{id}`  - raw transformed image
* `:4444/{site} `         - without scheme e.g. secretagentsnowman.com


