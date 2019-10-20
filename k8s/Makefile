
make: create-cluster kubectl deploy
	echo "Build and deploy."

create-cluster:
	eksctl create cluster \
	--name "cozyish" \
	--nodes 2 \
	--node-ami auto \
	--region=us-west-2

delete-cluster:
	eksctl delete cluster --name="cozyish"

kubectl:
	rm cozyish-k8s.yml
	kompose convert
	( for i in *.yaml ; do cat $$i ; echo '---' ; done ) >cozyish-k8s.yml
	rm -f *.yaml
	cat cozyish-k8s.yml | awk '{gsub(/extensions\/v1beta1/,"apps/v1")}1' > foo.yml
	mv foo.yml cozyish-k8s.yml

deploy:
	kubectl apply -f cozyish-k8s.yml

delete:
	kubectl delete -f cozyish-k8s.yml

