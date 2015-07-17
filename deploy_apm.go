package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

var ApmSeConfig = `{
    "kind": "Service",
    "apiVersion": "v1beta3",
    "metadata": {
        "name": "apm",
        "namespace": "default",
        "labels": {
            "name": "apm"
        }
    },
    "spec": {
        "ports": [
            {
                "protocol": "TCP",
                "port": 6669,
                "targetPort": 6669
            }
        ],
        "selector": {
            "name": "apm"
        }
    }
}`

var ApmRcConfig = `{
    "kind": "ReplicationController",
    "apiVersion": "v1beta3",
    "metadata": {
        "name": "apm",
        "namespace": "default",
        "labels": {
            "name": "apm"
        }
    },
    "spec": {
        "replicas": 1,
        "selector": {
            "name": "apm"
        },
        "template": {
            "metadata": {
                "labels": {
                    "name": "apm"
                }
            },
            "spec": {
                "containers": [
                    {
                        "name": "apm",
                        "image": "liuyang/apm-dc-master:v13",
                        "ports": [
                            {
                                "containerPort": 6669,
                                "protocol": "TCP"
                            }
                        ]
                    }
                ],
                "nodeSelector": {
                    "kubernetes.io/hostname": "127.0.0.1"
                }
            }
        }
    }
}`

func main() {

	MASTER := "127.0.0.1"
	client := &http.Client{}
	request, err := http.NewRequest("POST", "http://"+MASTER+":8080/api/v1beta3/namespaces/default/replicationcontrollers", strings.NewReader(ApmRcConfig))
	rcresp, err := client.Do(request)
	if err != nil {
		fmt.Println("new apm rc error:", err)
		return
	}
	defer rcresp.Body.Close()
	body, err := ioutil.ReadAll(rcresp.Body)
	fmt.Println(string(body))

	request, err = http.NewRequest("POST", "http://"+MASTER+":8080/api/v1beta3/namespaces/default/services", strings.NewReader(ApmSeConfig))
	seresp, err := client.Do(request)
	if err != nil {
		fmt.Println("new apm se error:", err)
		return
	}
	defer seresp.Body.Close()
	body, err = ioutil.ReadAll(seresp.Body)
	fmt.Println(string(body))
	//client := &http.Client{}
}
