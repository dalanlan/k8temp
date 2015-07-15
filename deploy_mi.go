package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
)

func main() {

	// check current user is root or not
	user, _ := user.Current()

	if user.Uid != "0" {
		fmt.Println("Please run as root")
		os.Exit(1)
	}

	startk8sScript := `install_k8s_minion() {
		
	K8S_VERSION=0.18.2
	PRIVATE_IP="10.168.10.5"
	HOSTNAME="10.168.10.5"
	USER=cxy
	DOCKER_CONF="/etc/default/docker"
	PRIVATE_PORT="5000"	
	MASTER_IP="10.168.14.145"
	lsb_dist="ubuntu"
	
	sudo -b docker -d -H unix:///var/run/docker-bootstrap.sock -p /var/run/docker-bootstrap.pid --iptables=false --ip-masq=false --graph=/var/lib/docker-bootstrap 2> /var/log/docker-bootstrap.log 1> /dev/null
	
	sudo docker -H unix:///var/run/docker-bootstrap.sock load -i f.tar	
     
	sleep 10
	
	# Start flannel
	flannelCID=$(sudo docker -H unix:///var/run/docker-bootstrap.sock run -d --net=host --privileged -v /dev/net:/dev/net wizardcxy/flannel:0.3.0 /opt/bin/flanneld --etcd-endpoints=http://${MASTER_IP}:4001 -iface="eth0")

	sleep 8
	rm -rf ./subnet.env
	#sudo docker -H unix:///var/run/docker-bootstrap.sock cp ${flannelCID}:/run/flannel/subnet.env .
	docker -H unix:///var/run/docker-bootstrap.sock exec ${flannelCID} cat /run/flannel/subnet.env > subnet.env
	source subnet.env

	# configure docker net settings and registry, then restart it
	echo "DOCKER_OPTS=\"\$DOCKER_OPTS --mtu=${FLANNEL_MTU} --bip=${FLANNEL_SUBNET} --insecure-registry ${USER}reg:5000\"" | sudo tee -a ${DOCKER_CONF}

	ifconfig docker0 down

    case "$lsb_dist" in
		fedora|centos)
            yum install bridge-utils && brctl delbr docker0 && systemctl restart docker
        ;;
        ubuntu|debian|linuxmint)
            apt-get install bridge-utils && brctl delbr docker0 && service docker restart
        ;;
    esac

	# sleep a little bit
	sleep 5

	echo "${MASTER_IP} ${USER}reg" | sudo tee -a /etc/hosts

	# Start minion
	sudo docker run --net=host -d -v /var/run/docker.sock:/var/run/docker.sock  wizardcxy/hyperkube:v${K8S_VERSION} /hyperkube kubelet --api_servers=http://${MASTER_IP}:8080 --v=2 --address=0.0.0.0 --enable_server --hostname_override=${HOSTNAME}
	sudo docker run -d --net=host --privileged wizardcxy/hyperkube:v${K8S_VERSION} /hyperkube proxy --master=http://${MASTER_IP}:8080 --v=2
	
	# Start cadvisor
	./monitor.sh

}
install_k8s_minion`
	// Install minion
	fmt.Println("Installing minion")
	cmd := exec.Command("bash", "-c", startk8sScript)
	res, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("minion installation done")
	fmt.Println(string(res))

}
