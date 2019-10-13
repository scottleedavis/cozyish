from urllib.request import urlopen
import urllib.request
import json   
from bs4 import BeautifulSoup
import re
from flask import Flask, jsonify
from flask_cors import CORS
from flask import request
from os import environ
import time

app = Flask(__name__)
app.config.from_object(__name__)

CORS(app, resources={r'/*': {'origins': '*'}})

API_URL = "127.0.0.1:8000"

max_depth = 2

@app.route('/', methods=['GET'])
def index():

	try:
		API_URL = environ['API']
	except:
		API_URL = "127.0.0.1:8000"

	all_urls = []
	url = request.args.get('url')

	all_urls.append(url)
	find_all_urls(url, url, all_urls)
	all_urls = list(set(all_urls))

	print(all_urls)

	all_images = find_all_images(all_urls)
	all_images = list(set(all_images))

	for image in all_images: 

		apiurl = "http://"+API_URL+"/api/index"
		req = urllib.request.Request(apiurl)
		req.add_header('Content-Type', 'application/json; charset=utf-8')
		jsondata = ""
		if image.startswith("http"):
			jsondata = json.dumps({"image": image})
		else:
			jsondata = json.dumps({"image": url+"/"+image})
		jsondataasbytes = jsondata.encode('utf-8')
		req.add_header('Content-Length', len(jsondataasbytes))
		urllib.request.urlopen(req, jsondataasbytes)

	print(all_images)

	return jsonify(all_images)


def find_all_images(all_urls):

	temp_images = []
	for url in all_urls:
		try:
			html = urlopen(url)
			bs = BeautifulSoup(html, 'html.parser')
			images = get_images(bs, url)
			# print("images:"+images)
			temp_images.extend(images)
		except:
			print("Error finding all images:"+url)

	return temp_images

def find_all_urls(root_url, url, all_urls,depth=1):

	if depth == max_depth:
		return

	temp_urls = []
	try:
		html = urlopen(url)
		bs = BeautifulSoup(html, 'html.parser')
		for a in bs.find_all('a', href=True):
			if a["href"].startswith(root_url) and a["href"] not in all_urls and a["href"] not in temp_urls:
				print ("Found the URL:", a['href'])
				temp_urls.append(a["href"])
			elif not a["href"].startswith("mailto") and not a["href"].startswith("http") and not a["href"].startswith("..") and root_url+"/"+a["href"] not in all_urls and root_url+a["href"] not in temp_urls and root_url+"/"+a["href"] not in temp_urls:
				if a["href"].startswith("/"):
					print ("Found the URL:", a["href"])
					temp_urls.append(root_url+a["href"])
				else:
					print ("Found the URL:", a["href"])
					temp_urls.append(root_url+"/"+a["href"])

	except:
		print("Error loading URL:"+url)

	for c in temp_urls:
		all_urls.append(c)
		find_all_urls(root_url, c, all_urls, depth+1)


def get_images(bs, url):
	images_output = []
	images = bs.find_all('img', {'src': lambda s: s.endswith((".jpg", ".jpeg", ".png"))})
	for image in images: 

		if image["src"].startswith("http"):
			images_output.append(image["src"])
		else:
			images_output.append(url+"/"+image["src"])

	return images_output

app.run(host='0.0.0.0', port=4444, debug=True)

