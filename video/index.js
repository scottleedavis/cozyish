const express = require('express')
const app = express()
const bodyParser= require('body-parser')
const multer = require('multer');
const extractFrames = require('ffmpeg-extract-frames')
const fs = require('fs')
const request = require('request');

const port = 3000
const storage = multer.diskStorage({
    destination: function (req, file, cb) {
      cb(null, 'uploads')
    },
    filename: function (req, file, cb) {
      cb(null, file.fieldname + '.webm')
    }
  })
const upload = multer({ storage: storage })
const VIDEO =  process.env.VIDEO ?  process.env.VIDEO : "localhost:3000"
const API   =  process.env.API ?  process.env.API : "localhost:8000"
app.use(bodyParser.urlencoded({extended: true}))

app.use(express.static('public'))

app.post('/api', upload.single('file'), async (req, res, next) => {
    const file = req.file
    if (!file) {
      const error = new Error('Please upload a file')
      error.httpStatusCode = 400
      return next(error)
  
    }
    const integers = Array(100);
    for (let x = 0; x < 100; x++) {
        integers[x] = x;
    }
    const offsets =  integers.map(function(x) { return x * 5000; });
    await extractFrames({
        input: 'uploads/file.webm',
        output: 'public/screenshots/screenshot-%i.jpg',
        offsets: offsets
      })

      var path = 'public/screenshots/'
 
      fs.readdir(path, function(err, items) {
          console.log(items);
       
          for (var i=0; i<items.length; i++) {
                const url = "http://"+VIDEO+"/screenshots/"+items[i];
                console.log(url);

                request('http://'+API+"/api/index", { 
                    json: {
                      image: url
                    }
                 }, (err, res, body) => {
                if (err) { return console.log(err); }
                console.log(body.url);
                console.log(body.explanation);
                });
          }
          res.send('{"status": "ok"}');

      });
    
});

app.listen(port, () => console.log(`Example app listening on port ${port}!`))