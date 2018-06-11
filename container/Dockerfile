FROM centos:7

USER root

MAINTAINER zkosic@redhat.com

RUN yum install -y epel-release && yum install -y jq-devel && yum install -y lshw && yum clean all

RUN mkdir -p /goroot && curl https://storage.googleapis.com/golang/go1.10.linux-amd64.tar.gz | tar xvzf - -C /goroot --strip-components=1

ENV GOROOT /goroot
ENV GOPATH /gopath
ENV PATH $GOROOT/bin:$GOPATH/bin:$PATH

WORKDIR /gopath/src/sfc-dev-plugin

COPY . .

RUN go build -o sfc-nic-plugin
RUN cp sfc-nic-plugin /usr/bin/sfc-nic-plugin && cp list_sfc_devices.sh /usr/bin/list_sfc_devices.sh


CMD ["sfc-nic-plugin", "-logtostderr=true"]
