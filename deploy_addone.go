package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func main() {
	loadScript := `loadpkg(){
	#load the file
	sudo docker load -i ./tarpackage/apm-dc-master.tar
	sudo docker load -i ./tarpackage/dnsImage/dnsetcd.tar
	sudo docker load -i ./tarpackage/dnsImage/dnsexec.tar
	sudo docker load -i ./tarpackage/dnsImage/dnskube2sky.tar
	sudo docker load -i ./tarpackage/dnsImage/dnsskydns.tar
	sleep 3
}
loadpkg
`

	var MASTER = ""
	flag.StringVar(&MASTER, "masterip", "", "input the master node ip (private)")

	flag.Parse()

	if MASTER == "" {
		fmt.Println("please input master ip")
		os.Exit(1)
	}

	//the json file to create se and pods
	var DnsRcConfig = `
{
    "apiVersion": "v1",
    "kind": "ReplicationController",
    "metadata": {
        "labels": {
            "k8s-app": "kube-dns",
            "kubernetes.io/cluster-service": "true",
            "version": "v8"
        },
        "name": "kube-dns-v8",
        "namespace": "default"
    },
    "spec": {
        "replicas": 1,
        "selector": {
            "k8s-app": "kube-dns",
            "version": "v8"
        },
        "template": {
            "metadata": {
                "labels": {
                    "k8s-app": "kube-dns",
                    "kubernetes.io/cluster-service": "true",
                    "version": "v8"
                }
            },
            "spec": {
                "containers": [
                    {
                        "command": [
                            "/usr/local/bin/etcd",
                            "-data-dir",
                            "/var/etcd/data",
                            "-listen-client-urls",
                            "http://127.0.0.1:2379,http://127.0.0.1:4001",
                            "-advertise-client-urls",
                            "http://127.0.0.1:2379,http://127.0.0.1:4001",
                            "-initial-cluster-token",
                            "skydns-etcd"
                        ],
                        "image": "gcr.io/google-containers/etcd:2.0.9",
                        "name": "etcd",
                        "resources": {
                            "limits": {
                                "cpu": "100m",
                                "memory": "50Mi"
                            }
                        },
                        "volumeMounts": [
                            {
                                "mountPath": "/var/etcd/data",
                                "name": "etcd-storage"
                            }
                        ]
                    },
                    {
                        "args": [
                            "-kube_master_url=http://` + MASTER + `:8080",
                            "-domain=cluster.local"
                        ],
                        "image": "dalanlan/kube2sky:1.11",
                        "name": "kube2sky",
                        "resources": {
                            "limits": {
                                "cpu": "100m",
                                "memory": "50Mi"
                            }
                        }
                    },
                    {
                        "args": [
                            "-machines=http://localhost:4001",
                            "-addr=0.0.0.0:53",
                            "-domain=cluster.local."
                        ],
                        "image": "gcr.io/google-containers/skydns:2015-03-11-001",
                        "livenessProbe": {
                            "httpGet": {
                                "path": "/healthz",
                                "port": 8080,
                                "scheme": "HTTP"
                            },
                            "initialDelaySeconds": 30,
                            "timeoutSeconds": 5
                        },
                        "name": "skydns",
                        "ports": [
                            {
                                "containerPort": 53,
                                "name": "dns",
                                "protocol": "UDP"
                            },
                            {
                                "containerPort": 53,
                                "name": "dns-tcp",
                                "protocol": "TCP"
                            }
                        ],
                        "resources": {
                            "limits": {
                                "cpu": "100m",
                                "memory": "50Mi"
                            }
                        }
                    },
                    {
                        "args": [
                            "-cmd=nslookup kubernetes.default.svc.cluster.local localhost >/dev/null",
                            "-port=8080"
                        ],
                        "image": "gcr.io/google_containers/exechealthz:1.0",
                        "name": "healthz",
                        "ports": [
                            {
                                "containerPort": 8080,
                                "protocol": "TCP"
                            }
                        ],
                        "resources": {
                            "limits": {
                                "cpu": "10m",
                                "memory": "20Mi"
                            }
                        }
                    }
                ],
                "dnsPolicy": "Default",
                "volumes": [
                    {
                        "emptyDir": {},
                        "name": "etcd-storage"
                    }
                ]
            }
        }
    }
}
`

	var DnsSeConfig = `
{
  "apiVersion": "v1",
  "kind": "Service",
  "metadata": {
    "name": "kube-dns",
    "namespace": "default",
    "labels": {
      "k8s-app": "kube-dns",
      "kubernetes.io/cluster-service": "true",
      "kubernetes.io/name": "KubeDNS"
    }
  },
  "spec": {
    "selector": {
      "k8s-app": "kube-dns"
    },
    "clusterIP": "192.168.3.10",
    "ports": [
      {
        "name": "dns",
        "port": 53,
        "protocol": "UDP"
      },
      {
        "name": "dns-tcp",
        "port": 53,
        "protocol": "TCP"
      }
    ]
  }
}

`

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
                ]
            }
        }
    }
}`

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

	fmt.Println("load the tar file")
	cmd := exec.Command("bash", "-c", loadScript)
	//cmd := exec.Command("bash", "-c", "echo ok")
	res, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("load operation done")
	fmt.Println(string(res))

	client := &http.Client{}

	//create DNS rn and DNS service
	request, err := http.NewRequest("POST", "http://"+MASTER+":8080/api/v1/namespaces/default/replicationcontrollers", strings.NewReader(DnsRcConfig))
	dnsrcresp, err := client.Do(request)
	if err != nil {
		fmt.Println("new dns rc error:", err)
		return
	}
	defer dnsrcresp.Body.Close()
	body, err := ioutil.ReadAll(dnsrcresp.Body)
	fmt.Println(string(body))

	request, err = http.NewRequest("POST", "http://"+MASTER+":8080/api/v1/namespaces/default/services", strings.NewReader(DnsSeConfig))
	dnsseresp, err := client.Do(request)
	if err != nil {
		fmt.Println("new dns se error:", err)
		return
	}
	defer dnsseresp.Body.Close()
	body, err = ioutil.ReadAll(dnsseresp.Body)
	fmt.Println(string(body))

	//create apm-dc rc and se
	request, err = http.NewRequest("POST", "http://"+MASTER+":8080/api/v1beta3/namespaces/default/replicationcontrollers", strings.NewReader(ApmRcConfig))
	rcresp, err := client.Do(request)
	if err != nil {
		fmt.Println("new apm rc error:", err)
		return
	}
	defer rcresp.Body.Close()
	body, err = ioutil.ReadAll(rcresp.Body)
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

}
