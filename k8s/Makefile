
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

deploy:
	kubectl apply -f cozyish.yml

delete:
	kubectl delete -f cozyish.yml

