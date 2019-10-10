#!/bin/sh

rm cozyish-k8s.yml

kompose convert

( for i in *.yaml ; do cat $i ; echo '---' ; done ) >cozyish-k8s.yml

rm -f *.yaml

echo "Deploy to kubernetes `kubectl apply -f cozyish-k8s.yml`"
