
make: create-cluster kubectl deploy
	echo "Build and deploy."

create-cluster:
	eksctl create cluster \
	--name "cozyish2" \
	--nodes 2 \
	--node-ami auto \
	--region=us-west-2

delete-cluster:
	eksctl delete cluster --name="cozyish2"

dashboard:
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.0.0-beta4/aio/deploy/recommended.yaml
	kubectl apply -f dashboard.yml
	kubectl -n kubernetes-dashboard describe secret $(kubectl -n kubernetes-dashboard get secret | grep admin-user | awk '{print $1}')
	echo "TODO:  Find admin-user-token."
	echo "TODO: run kubectl proxy"
	echo "TODO: open http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/"

deploy:
	kubectl apply -f cozyish.yml

delete:
	kubectl delete -f cozyish.yml

