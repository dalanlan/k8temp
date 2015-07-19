package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

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
		if strings.Contains(IP, "/24") {
			masterip = strings.TrimSuffix(IP, "/24")
			log.Printf("the master ip of %v is : %v \n", ifac, masterip)

		}
	}

	return masterip, nil

}

func checkpara(MASTERIP, IFACE, UNAME string) (bool, error) {
	if MASTERIP == "" {
		return false, errors.New("please input the master ip")
	}

	if UNAME == "" {
		return false, errors.New("please input the username")
	}

	return true, nil

}

func main() {

	//check current user is root or not
	user, _ := user.Current()

	if user.Uid != "0" {
		log.Println("Please run as root")
		os.Exit(1)
	}

	//get the username and password and net interface
	MASTERIP := ""
	IFACE := ""
	UNAME := ""

	flag.StringVar(&MASTERIP, "master", "", "input the master ip (private)")
	flag.StringVar(&IFACE, "iface", "eth0", "input the net interface default eth0")
	flag.StringVar(&UNAME, "u", "", "input the username")

	privateip, err := checkip(IFACE)
	flag.Parse()

	if err != nil {
		log.Println(err.Error())
	}

	checkok, err := checkpara(MASTERIP, IFACE, UNAME)

	if !checkok {
		log.Println("installation fail:")
		log.Println(err.Error())
		return
	}

	startk8sScript := `install_k8s_minion() {
		
	K8S_VERSION=0.18.2
	PRIVATE_IP=` + privateip + `
	HOSTNAME=` + privateip + `
	USER=` + UNAME + `
	PRIVATE_PORT="5000"	
	IFACE=` + IFACE + `
	MASTER_IP=` + MASTERIP + `
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
	
	
	sudo -b docker -d -H unix:///var/run/docker-bootstrap.sock -p /var/run/docker-bootstrap.pid --iptables=false --ip-masq=false --graph=/var/lib/docker-bootstrap 2> /var/log/docker-bootstrap.log 1> /dev/null
	
	sudo docker -H unix:///var/run/docker-bootstrap.sock load -i ./tarpackage/f.tar
	sudo docker -H unix:///var/run/docker-bootstrap.sock load -i ./tarpackage/hyperbase.tar

     
	sleep 10
	
	# Start flannel
	flannelCID=$(sudo docker -H unix:///var/run/docker-bootstrap.sock run -d --net=host --privileged -v /dev/net:/dev/net wizardcxy/flannel:0.3.0 /opt/bin/flanneld --etcd-endpoints=http://${MASTER_IP}:4001 -iface=${IFACE})

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

	if grep -Fxq "${PRIVATE_IP} ${USER}reg" /etc/hosts
	then
	echo "modify /etc/hosts"
	else
	echo "${PRIVATE_IP} ${USER}reg" | sudo tee -a /etc/hosts
	fi

	# Start minion
	sudo docker run --net=host -d -v /var/run/docker.sock:/var/run/docker.sock  wizardcxy/hyperkube:v${K8S_VERSION} /hyperkube kubelet --api_servers=http://${MASTER_IP}:8080 --v=2 --address=0.0.0.0 --enable_server --hostname_override=${HOSTNAME}
	sudo docker run -d --net=host --privileged wizardcxy/hyperkube:v${K8S_VERSION} /hyperkube proxy --master=http://${MASTER_IP}:8080 --v=2
	
	# Start cadvisor
	
	sudo docker load -i ./tarpackage/monitserver.tar
    sudo docker load -i ./tarpackage/cadvisor.tar
	
	    sudo docker run \
      --volume=/:/rootfs:ro \
      --volume=/var/run:/var/run:rw \
      --volume=/sys:/sys:ro \
      --volume=/var/lib/docker/:/var/lib/docker:ro \
      --publish=4194:8080 \
      --detach=true \
    google/cadvisor:latest

    sleep 3

    sudo docker run --net=host --privileged -d monitserver:latest

    echo "Monitserver installation ok"

}
install_k8s_minion`
	// Install minion
	log.Println("Installing minion")
	cmd := exec.Command("bash", "-c", startk8sScript)
	res, err := cmd.Output()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("minion installation done")
	log.Println(string(res))

}
