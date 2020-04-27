#!/usr/bin/env bash

while :
do
	curl http://localhost:8080/
	sleep $(($RANDOM % 10))
done
