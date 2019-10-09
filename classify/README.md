# cozyish-classify

```bash
go run main.go
```

### Notes

https://www.deepdetect.com/server/docs/imagenet-classifier/

docker run -d -p 8080:8080 -v /Users/scottd/workspace/cozyish/classify/models:/opt/models/ jolibrain/deepdetect_cpu

load pretrained model
curl -X PUT 'http://localhost:8080/services/imageserv' -d '{
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
}
'

others trained models:
https://www.deepdetect.com/models/?opts={%22media%22:%22image%22,%22type%22:%22type-all%22,%22backend%22:[%22caffe%22,%22tensorflow%22,%22caffe2%22],%22platform%22:%22desktop%22,%22searchTerm%22:%22%22}
