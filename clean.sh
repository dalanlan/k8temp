#!/bin/bash

sudo docker rm -f $(docker ps -a -q)

sudo docker -H unix:///var/run/docker-bootstrap.sock rm -f $(docker -H unix:///var/run/docker-bootstrap.sock ps -a -q)

# flannel port
kill -9 $(lsof -i :8285 | awk '{print $2}')
# clean cadvisor
kill -9 $(lsof -i :4194 | awk '{print $2}')

kill -9 $(ps -ef|grep docker |grep -v grep |awk '{print $2}')

# clean 8080
kill -9 $(lsof -i :80 | awk '{print $2}')

rm -f /etc/default/docker

sudo service docker restart
