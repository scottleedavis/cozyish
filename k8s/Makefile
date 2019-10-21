
make: create-cluster kubectl deploy
	echo "Build and deploy."

create-cluster:
	eksctl create cluster \
	--name "cozyish-cluster" \
	--nodes 2 \
	--node-ami auto \
	--region=us-west-2

delete-cluster:
	eksctl delete cluster --name="cozyish-cluster"

kubectl:
	rm -f cozyish.yml
	( for i in *.yaml ; do cat $$i ; echo '---' ; done ) >cozyish.yml

deploy:
	kubectl apply -f elastic.yaml
	kubectl apply -f minio.yaml
	kubectl apply -f rabbit.yaml
	kubectl apply -f nsfw.yaml
	kubectl apply -f deep.yaml

	kubectl apply -f ui.yaml
	kubectl apply -f crawler.yaml
	kubectl apply -f api.yaml
	kubectl apply -f store.yaml
	kubectl apply -f extract.yaml
	kubectl apply -f classify.yaml
	kubectl apply -f cache.yaml

delete:
	kubectl delete -f elastic.yaml
	kubectl delete -f minio.yaml
	kubectl delete -f rabbit.yaml
	kubectl delete -f nsfw.yaml
	kubectl delete -f deep.yaml

	kubectl delete -f ui.yaml
	kubectl delete -f crawler.yaml
	kubectl delete -f api.yaml
	kubectl delete -f store.yaml
	kubectl delete -f extract.yaml
	kubectl delete -f classify.yaml
	kubectl delete -f cache.yaml

