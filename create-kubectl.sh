#!/bin/sh

rm cozyish-k8s.yml

kompose convert

( for i in *.yaml ; do cat $i ; echo '---' ; done ) >cozyish-k8s.yml

rm -f *.yaml

cat cozyish-k8s.yml | awk '{gsub(/extensions\/v1beta1/,"apps/v1")}1' > foo.yml
mv foo.yml cozyish-k8s.yml

echo "Deploy to kubernetes: \"kubectl apply -f cozyish-k8s.yml\""
