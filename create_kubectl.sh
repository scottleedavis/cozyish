#!/bin/sh

kompose convert
mv *.yaml kubernetes

#kubectl apply -f $(`ls -m kubernetes/ | sed -e 's/, /,/g'`)
