package main

import (
	//"bytes"
	"crypto/tls"
	//"crypto/x509"
	//"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func login(user, pw, masterip, cacertLoc, clustername string) (bool, error) {

	if user == "" || pw == "" {

		return false, errors.New("please input the user name and password use -h to get detail info")

	}

	if masterip == "" {
		return false, errors.New("please input the public master ip use -h to get detail info")
	}

	if clustername == "" {
		return false, errors.New("please input the public cluster name use -h to get detail info")
	}

	caCertPath := cacertLoc // load ca file
	caCrt, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		log.Println("ReadFile err:", err)
		return false, nil
	}

	// use map as struct
	var clusterinfo = url.Values{}
	//var clusterinfo = map[string]string{}
	clusterinfo.Add("userName", user)
	clusterinfo.Add("password", pw)
	clusterinfo.Add("cloudName", clustername)
	clusterinfo.Add("masterIp", masterip)
	clusterinfo.Add("cacrt", string(caCrt))

	data := clusterinfo.Encode()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true},
		DisableCompression: true,
	}

	client := &http.Client{Transport: tr}

	url := "https://183.129.190.82:9000/user/checkAndUpdate"
	//url := "https://10.10.105.124:8443/user/checkAndUpdate"
	reqest, err := http.NewRequest("POST", url, strings.NewReader(data))
	reqest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqest.Header.Set("Authorization", "qwertyuiopasdfghjklzxcvbnm1234567890")
	resp, err := client.Do(reqest)

	if err != nil {
		//panic(err)
		return false, err
	}

	body, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(body))

	if resp.StatusCode == 400 {
		return false, errors.New("cluster name auth fail")
	}

	if resp.StatusCode == 200 {
		return true, nil
	}

	return false, errors.New("auth fail")
}

func checkip(ifac string) (string, error) {
	ifacobj, err := net.InterfaceByName(ifac)
	if err != nil {
		//log.Println(err.Error())
		return "", err
	}

	addrarry, err := ifacobj.Addrs()
	if err != nil {
		//log.Println(err.Error())
		return "", err
	}

	//log.Println(addrarry)
	var masterip = ""
	for _, ip := range addrarry {
		IP := ip.String()
		log.Println(IP)
		//  the ipv4 may return xx.xx.xx.xx/2*
		if strings.Contains(IP, "/") && !strings.Contains(IP, ":") {
			masterip = strings.Split(IP, "/")[0]
			log.Printf("the master ip is : %v \n", masterip)

		}
	}

	return masterip, nil

}

func createca(username string, cluster string) error {
	var genca = `
	cd ./cert
	rm *
	openssl genrsa -out ca.key 2048

    openssl req -x509 -new -nodes -key ca.key -subj "/CN=zju.com" -days 5000 -out ca.crt

    openssl genrsa -out server.key 2048

    openssl req -new -key server.key -subj "/CN=` + cluster + "." + username + `" -out server.csr

    openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 5000
	
	touch tokens.csv
	
	echo "abcdTOKEN1234,zju,zju" > tokens.csv
	
	cd ..
	
	`
	cmd := exec.Command("bash", "-c", genca)
	res, err := cmd.Output()
	if err != nil {
		return errors.New("fail to create ca file , please installing openssl")
	}

	log.Println(string(res))

	return nil
}

func main() {
	//check current user is root or not
	user, _ := user.Current()

	if user.Uid != "0" {
		log.Println("Please run as root")
		os.Exit(1)
	}

	//get the username and password and net interface
	var iface, pass, uname, publicip, cluster string
	flag.StringVar(&iface, "iface", "eth0", "input the net interface default eth0")
	flag.StringVar(&pass, "p", "", "input the password")
	flag.StringVar(&uname, "u", "", "input the username")
	flag.StringVar(&publicip, "publicip", "", "input the public ip")
	flag.StringVar(&cluster, "cluster", "", "input the cluster name")

	//use flag?
	certLoc := "cert/ca.crt"
	flag.Parse()

	userName := uname
	pw := pass
	//todo get the masterip when checkuser
	masterprivateip, err := checkip(iface)
	log.Println("private ip :", masterprivateip)
	masterpublicip := publicip
	if err != nil {
		log.Println("installation fail :")
		log.Println(err.Error())
		os.Exit(1)
	}

	logcheck, err := login(userName, pw, masterpublicip, certLoc, cluster)
	log.Println(userName, pw, masterpublicip, certLoc, cluster)
	//logcheck := true
	if !logcheck {
		log.Println("installation fail :")
		if err != nil {
			log.Println(err.Error())
		}

		os.Exit(1)
	}

	log.Println("valid user and password, continue")

	USER := string(userName)
	PRIVATEIP := string(masterprivateip)
	IFACE := string(iface)
	//the cn of ca.crt is clustername.username
	err = createca(USER, cluster)
	if err != nil {
		log.Println("installation fail :")
		log.Println(err.Error())
	}

	var startk8sScript = `start_k8s(){
	K8S_VERSION=0.18.2
	# attention to modify master ip to public ip and cert path
	#PRIVATE_IP=10.168.14.145
	PRIVATE_IP=` + PRIVATEIP + `
	USER=` + USER + `
	PRIVATE_PORT="5000"
	IFACE=` + IFACE + `
	lsb_dist="$(lsb_release -si)"
	
	lsb_dist="$(echo "$lsb_dist" | tr '[:upper:]' '[:lower:]')"

    case "$lsb_dist" in
		fedora|centos)
            DOCKER_CONF="/etc/sysconfig/docker"
        ;;
        ubuntu|debian|linuxmint)
            DOCKER_CONF="/etc/default/docker"
        ;;
    esac
	
	
	#CERT_PATH="/root/wangzhe/kube-in-docker/cert"
	HOSTDIR="/mnt"
    LOCAL_PATH=$(pwd)
	#sudo -b docker -d -H unix:///var/run/docker-bootstrap.sock -p /var/run/docker-bootstrap.pid --iptables=false --ip-masq=false --bridge=none --graph=/var/lib/docker-bootstrap 2> /var/log/docker-bootstrap.log 1> /dev/null
	sudo -b docker -d -H unix:///var/run/docker-bootstrap.sock -p /var/run/docker-bootstrap.pid --iptables=false --ip-masq=false --graph=/var/lib/docker-bootstrap 2> /var/log/docker-bootstrap.log 1> /dev/null
	sleep 5
	sudo docker -H unix:///var/run/docker-bootstrap.sock load -i ./tarpackage/f.tar
	sudo docker -H unix:///var/run/docker-bootstrap.sock load -i ./tarpackage/e.tar
	
	sudo docker load -i ./tarpackage/hyperbase.tar
	sudo docker load -i ./tarpackage/r.tar
	sudo docker load -i ./tarpackage/p.tar
	sudo docker load -i ./tarpackage/g.tar
	sudo docker load -i ./tarpackage/b.tar
	sudo docker load -i ./tarpackage/apiserver.tar
	sudo docker load -i ./tarpackage/appbase.tar
	
	 # Start etcd
	docker -H unix:///var/run/docker-bootstrap.sock run --restart=on-failure:10 --net=host -d wizardcxy/etcd:2.0.9 /usr/local/bin/etcd --addr=${PRIVATE_IP}:4001 --bind-addr=0.0.0.0:4001 --data-dir=/var/etcd/data
	sleep 5
	# Set flannel net config
	docker -H unix:///var/run/docker-bootstrap.sock run --net=host wizardcxy/etcd:2.0.9 etcdctl set /coreos.com/network/config '{ "Network": "10.1.0.0/16" }'
	# iface may change to a private network interface, eth0 is for ali ecs
	sleep 2
	flannelCID=$(docker -H unix:///var/run/docker-bootstrap.sock run -d --net=host --privileged -v /dev/net:/dev/net quay.io/coreos/flannel:0.3.0 /opt/bin/flanneld -iface=${IFACE})
	sleep 6
	# Configure docker net settings and registry setting and restart it
	#docker -H unix:///var/run/docker-bootstrap.sock cp ${flannelCID}:/run/flannel/subnet.env .
	docker -H unix:///var/run/docker-bootstrap.sock exec ${flannelCID} cat /run/flannel/subnet.env > subnet.env
	source subnet.env
	# use insecure docker DOCKER_CONF ???registry ? write where??
	echo "DOCKER_OPTS=\"\$DOCKER_OPTS -H=unix:///var/run/docker.sock -H tcp://0.0.0.0:2376 --mtu=${FLANNEL_MTU} --bip=${FLANNEL_SUBNET} --insecure-registry=${USER}reg:${PRIVATE_PORT}\"" | sudo tee -a ${DOCKER_CONF}
	ifconfig docker0 down
	
	
	case "$lsb_dist" in
		fedora|centos)
            yum install bridge-utils && brctl delbr docker0 && systemctl restart docker
        ;;
        ubuntu|debian|linuxmint)
            apt-get install bridge-utils && brctl delbr docker0 && service docker restart
        ;;
    esac
	
	
	sleep 5
	docker run --restart=on-failure:10 -itd -p 5000:5000 -v ${HOSTDIR}:/tmp/registry-dev wizardcxy/registry:2.0
	if grep -Fxq "${PRIVATE_IP} ${USER}reg" /etc/hosts
	then
	echo "modify /etc/hosts"
	else
	echo "${PRIVATE_IP} ${USER}reg" | sudo tee -a /etc/hosts
	fi
	docker run --net=host --restart=on-failure:10 -itd -p 80:8081 -p 8082 liuyilun/gorouter
	#start api server (attention to the certpath)
	sudo docker load -i apiserver.tar
	# using private ip to comunicate with 8080
	#???
	docker run -id --restart=on-failure:10 --net=host  -v ${LOCAL_PATH}/cert:/cert/ apiserver:v1 /kube-apiserver --insecure-bind-address=${PRIVATE_IP} --insecure-port=8080 --bind-address=0.0.0.0 --secure-port=8081 --etcd_servers=http://${PRIVATE_IP}:4001 --logtostderr=true --service-cluster-ip-range=192.168.3.0/24 --token_auth_file=/cert/tokens.csv --client_ca_file=/cert/ca.crt --tls-private-key-file=/cert/server.key --tls-cert-file=/cert/server.crt
	#docker run -id --restart=on-failure:10 --net=host  -v ${LOCAL_PATH}/cert:/cert/ apiserver:v1 /kube-apiserver --insecure-bind-address=${PRIVATE_IP} --insecure-port=8080 --bind-address=0.0.0.0 --secure-port=8081 --etcd_servers=http://${PRIVATE_IP}:4001 --logtostderr=true --service-cluster-ip-range=192.168.3.0/24 --token_auth_file=/cert/tokens.csv --client_ca_file=/cert/ca.crt --tls-private-key-file=/cert/server.key --tls-cert-file=/cert/server.crt
	sleep 5
	# Start Master components (two add start policy) attention dns config dns ip could be assigned manually
	
	rm ./image/master-two.json
	cp ./image/master-two-template.json ./image/master-two.json
	sed -i "s/PRIVATEIP/${PRIVATE_IP}/g" ./image/master-two.json
	docker run --net=host -d -v /var/run/docker.sock:/var/run/docker.sock  -v ${LOCAL_PATH}/image/master-two.json:/etc/kubernetes/manifests-two/master.json  wizardcxy/hyperkube:v${K8S_VERSION} /hyperkube kubelet --api_servers=http://${PRIVATE_IP}:8080 --v=2 --address=${PRIVATE_IP} --enable_server --hostname_override=${PRIVATE_IP} --config=/etc/kubernetes/manifests-two --cluster_dns=192.168.3.10 --cluster_domain=cluster.local
	sleep 3
	docker run -d --net=host --privileged wizardcxy/hyperkube:v${K8S_VERSION} /hyperkube proxy --master=http://${PRIVATE_IP}:8080 --v=2
	# Start Monitor
	sudo docker load -i ./tarpackage/monitserver.tar
    sudo docker load -i ./tarpackage/cadvisor.tar

    sleep 5

    sudo docker run \
      --volume=/:/rootfs:ro \
      --volume=/var/run:/var/run:rw \
      --volume=/sys:/sys:ro \
      --volume=/var/lib/docker/:/var/lib/docker:ro \
      --publish=4194:8080 \
      --detach=true \
    google/cadvisor:latest

sleep 3

    sudo docker run --privileged=true --net=host  -d -v '/etc/ssl/certs:/etc/ssl/certs' monitserver:latest

    echo "Monitserver installation ok"
	
	
	}
	start_k8s`

	//only in ubuntu now...
	//Install master
	log.Println("Installing master")

	cmd := exec.Command("bash", "-c", startk8sScript)

	res, err := cmd.Output()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("Installation done")
	log.Println(string(res))

}
