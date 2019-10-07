from urllib.request import urlopen
from bs4 import BeautifulSoup
import re
from flask import Flask, jsonify
from flask_cors import CORS

app = Flask(__name__)
app.config.from_object(__name__)

CORS(app, resources={r'/*': {'origins': '*'}})

@app.route('/<url>', methods=['GET'])
def index(url):
	images_output = []
	try:
		html = urlopen("https://"+url)
		bs = BeautifulSoup(html, 'html.parser')
		images = bs.find_all('img', {'src':re.compile('.jpg')})
		for image in images: 
			print(image['src']+'\n')
			images_output.append(image['src'])
	except:
		html = urlopen("http://"+url)
		bs = BeautifulSoup(html, 'html.parser')
		images = bs.find_all('img', {'src':re.compile('.jpg')})
		for image in images: 
			print(image['src']+'\n')
			images_output.append(image['src'])

	return jsonify(images_output)

app.run(host='0.0.0.0', port=5000, debug=True)

