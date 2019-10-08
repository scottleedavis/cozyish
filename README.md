# cozyish

_todo_
* extract all docker-compose urls to environment variables
* classify
* fun learning

![](cozyish.png)


### Running
```bash
docker-compose build

docker-compose up
```

### Concept

A crawler image searches a provided site url for all png & jpg images, and index's them against the api.  The index payload is 
```json
{
    "image": "http://example.com/some/image/path.jpg"
    "tags": ["optional", "tags"]
}

Three endpoints exist on the api
* `:8000/api/index`     - indexes given the above payload, returns the payload + a generated id field.
* `:8000/api/image`     - json array of indexed/stored/transformed images
* `:8000/api/image/{id} - raw transformed image
* `:4444/{site} - without scheme e.g. secretagentsnowman.com


