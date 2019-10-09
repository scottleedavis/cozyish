from urllib.request import urlopen
import urllib.request
import json   
from bs4 import BeautifulSoup
import re
from flask import Flask, jsonify
from flask_cors import CORS
from flask import request
from os import environ

app = Flask(__name__)
app.config.from_object(__name__)

CORS(app, resources={r'/*': {'origins': '*'}})

API_URL = "127.0.0.1:8000"

@app.route('/', methods=['GET'])
def index():

	try:
		API_URL = environ['API']
	except:
		API_URL = "127.0.0.1:8000"

	images_output = []
	url = request.args.get('url')
	
	myurl = "http://"+API_URL+"/api/index"
	req = urllib.request.Request(myurl)
	html = urlopen(url)
	bs = BeautifulSoup(html, 'html.parser')
	images = bs.find_all('img', {'src': lambda s: s.endswith((".jpg", ".jpeg", ".png"))})
	for image in images: 

		if image["src"].startswith("http"):
			images_output.append(image["src"])
		else:
			images_output.append(url+"/"+image["src"])

		req.add_header('Content-Type', 'application/json; charset=utf-8')
		jsondata = json.dumps({"tags": [], "image": url+"/"+image["src"]})
		jsondataasbytes = jsondata.encode('utf-8')   # needs to be bytes
		req.add_header('Content-Length', len(jsondataasbytes))
		urllib.request.urlopen(req, jsondataasbytes)
		
	return jsonify(images_output)

app.run(host='0.0.0.0', port=4444, debug=True)

