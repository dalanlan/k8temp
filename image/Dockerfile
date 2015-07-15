FROM google/debian:wheezy

RUN apt-get update
RUN apt-get -yy -q install iptables ca-certificates
COPY hyperkube /hyperkube
RUN chmod a+rx /hyperkube
RUN mkdir cert
RUN mkdir -p /etc/kubernetes/manifests-two
RUN mkdir -p /etc/kubernetes/manifests
RUN mkdir -p /etc/kubernetes/manifests-https


#COPY master-https.json /etc/kubernetes/manifests-https/master.json
#COPY master.json /etc/kubernetes/manifests/master.json
#COPY master-two.json /etc/kubernetes/manifests-two/master.json