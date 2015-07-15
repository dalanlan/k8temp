package main

import (
	//	"bytes"
	//	"crypto/tls"
	//	"crypto/x509"
	//	"encoding/json"
	"fmt"
	//	"io/ioutil"
	//	"net/http"
	"os"
	"os/exec"
	"os/user"
)

func login(user, pw, masterip, cacertLoc string) bool {
	//pool := x509.NewCertPool()

	//caCertPath := cacertLoc // load ca file
	//caCrt, err := ioutil.ReadFile(caCertPath)
	//if err != nil {
	//	fmt.Println("ReadFile err:", err)
	//	return false
	//}
	//pool.AppendCertsFromPEM(caCrt)

	//tr := &http.Transport{
	//	TLSClientConfig:    &tls.Config{RootCAs: pool},
	//	DisableCompression: true,
	//}
	//client := &http.Client{Transport: tr}

	//// use map as struct
	//var clusterinfo = map[string]string{}
	//clusterinfo["username"] = user
	//clusterinfo["password"] = pw
	//clusterinfo["masterip"] = masterip
	//clusterinfo["cacrt"] = string(caCrt)
	//body, _ := json.Marshal(clusterinfo)
	//resp, err := client.Post("https://Apitransfer:10443/v1/application/checkuser", "application/json", bytes.NewReader(body))
	//if err != nil {
	//	//panic(err)
	//}

	//body, _ = ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))

	////return resp.StatusCode == 200
	return true
}
func main() {
	// check current user is root or not
	user, _ := user.Current()

	if user.Uid != "0" {
		fmt.Println("Please run as root")
		os.Exit(1)
	}

	userName := "yyy"
	pw := "123456"
	masterip := "10.10.103.250"
	certLoc := "ca.crt"
	if !login(userName, pw, masterip, certLoc) {
		fmt.Println("invalid username or password")
		os.Exit(1)
	}
	fmt.Println("valid user and password, continue")
	//only in ubuntu...
	startk8sScript := `start_k8s(){
	K8S_VERSION=0.18.2
	PRIVATE_IP=10.10.101.100
	USER=cxy
	DOCKER_CONF="/etc/default/docker"
	PRIVATE_PORT="5000"
    
	sudo rm -rf /var/lib/docker-bootstrap
	sudo rm /var/run/docker-bootstrap.pid 

	
	#sudo -b docker -d -H unix:///var/run/docker-bootstrap.sock -p /var/run/docker-bootstrap.pid --iptables=false --ip-masq=false --bridge=none --graph=/var/lib/docker-bootstrap 2> /var/log/docker-bootstrap.log 1> /dev/null
	sudo -b docker -d -H unix:///var/run/docker-bootstrap.sock -p /var/run/docker-bootstrap.pid --iptables=false --ip-masq=false --graph=/var/lib/docker-bootstrap 2> /var/log/docker-bootstrap.log 1> /dev/null

	sleep 5
	sudo docker -H unix:///var/run/docker-bootstrap.sock load -i f.tar
	sudo docker -H unix:///var/run/docker-bootstrap.sock load -i e.tar
	sudo docker -H unix:///var/run/docker-bootstrap.sock load -i monitserver-mini.tar
	sudo docker load -i h.tar
	sudo docker load -i r.tar
	sudo docker load -i p.tar
	sudo docker load -i g.tar
    sudo docker load -i b.tar
	
    # Start etcd
	docker -H unix:///var/run/docker-bootstrap.sock run --net=host -d wizardcxy/etcd:2.0.9 /usr/local/bin/etcd --addr=127.0.0.1:4001 --bind-addr=0.0.0.0:4001 --data-dir=/var/etcd/data

	sleep 5
	# Set flannel net config
	docker -H unix:///var/run/docker-bootstrap.sock run --net=host wizardcxy/etcd:2.0.9 etcdctl set /coreos.com/network/config '{ "Network": "10.1.0.0/16" }'
    
    # iface may change to a private network interface, eth0 is for ali ecs
    flannelCID=$(docker -H unix:///var/run/docker-bootstrap.sock run -d --net=host --privileged -v /dev/net:/dev/net quay.io/coreos/flannel:0.3.0 /opt/bin/flanneld -iface="eth0")
	
	sleep 8

	# Configure docker net settings and registry setting and restart it
	#docker -H unix:///var/run/docker-bootstrap.sock cp ${flannelCID}:/run/flannel/subnet.env .
	docker -H unix:///var/run/docker-bootstrap.sock exec ${flannelCID} cat /run/flannel/subnet.env > subnet.env
	source subnet.env

    # use insecure docker DOCKER_CONF ???registry ? write where??
	echo "DOCKER_OPTS=\"\$DOCKER_OPTS --mtu=${FLANNEL_MTU} --bip=${FLANNEL_SUBNET} --insecure-registry=${USER}reg:${PRIVATE_PORT}\"" | sudo tee -a ${DOCKER_CONF}

	ifconfig docker0 down

	apt-get install bridge-utils && brctl delbr docker0 && service docker restart

	sleep 5

	docker run --restart=on-failure:10 -itd -p 5000:5000 -v ${HOSTDIR}:/tmp/registry-dev wizardcxy/registry:2.0
    
    echo "${PRIVATE_IP} ${USER}reg" | sudo tee -a /etc/hosts

    docker run --restart=on-failure:10 -itd -p 80:8081 -p 8082 liuyilun/gorouter

    #start api server (attention to the certpath)
	sudo docker load -i apiserver.tar
	
	docker run  -it --net=host  -v /root/kube-in-docker/cert:/cert/ apiserver:v1 /kube-apiserver --insecure-bind-address=0.0.0.0 --insecure-port=8080 --bind-address=0.0.0.0 --secure-port=8081 --etcd_servers=http://127.0.0.1:4001 --logtostderr=true --service-cluster-ip-range=192.168.3.0/24 --token_auth_file=/cert/tokens.csv --client_ca_file=/cert/ca.crt --tls-private-key-file=/cert/server.key --tls-cert-file=/cert/server.crt

    # Start Master components (two)
	docker run --net=host -d -v /var/run/docker.sock:/var/run/docker.sock -v /root/kube-in-docker/cert:/cert/ wizardcxy/hyperkube:v${K8S_VERSION} /hyperkube kubelet --api_servers=http://localhost:8080 --v=2 --address=0.0.0.0 --enable_server --hostname_override=127.0.0.1 --config=/etc/kubernetes/manifests-two
    
	docker run -d --net=host --privileged wizardcxy/hyperkube:v${K8S_VERSION} /hyperkube proxy --master=http://127.0.0.1:8080 --v=2  
	
	# Start Monitor
	./monitor.sh
	
	
	
}
start_k8s`

	// Install master
	fmt.Println("Installing master")
	cmd := exec.Command("bash", "-c", startk8sScript)
	res, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Installation done")
	fmt.Println(string(res))

}
