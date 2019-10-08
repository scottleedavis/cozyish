from urllib.request import urlopen
import urllib.request
import json   
from bs4 import BeautifulSoup
import re
from flask import Flask, jsonify
from flask_cors import CORS

app = Flask(__name__)
app.config.from_object(__name__)

CORS(app, resources={r'/*': {'origins': '*'}})

API_URL = "api:8000"

@app.route('/<url>', methods=['GET'])
def index(url):
	images_output = []

	myurl = "http://"+API_URL+"/api/index"
	req = urllib.request.Request(myurl)

	scheme = 'https://'
	try:
		html = urlopen(scheme+url)
	except:
		scheme = 'http://'
		html = urlopen(scheme+url)

	bs = BeautifulSoup(html, 'html.parser')
	images = bs.find_all('img', {'src': lambda s: s.endswith((".jpg", ".jpeg", ".png"))})
	for image in images: 
		images_output.append(scheme+url+"/"+image["src"])
		req.add_header('Content-Type', 'application/json; charset=utf-8')
		jsondata = json.dumps({"id": 1, "image": scheme+url+"/"+image["src"]})
		jsondataasbytes = jsondata.encode('utf-8')   # needs to be bytes
		req.add_header('Content-Length', len(jsondataasbytes))
		urllib.request.urlopen(req, jsondataasbytes)
		
	return jsonify(images_output)

app.run(host='0.0.0.0', port=4444, debug=True)

