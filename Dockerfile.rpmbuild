FROM centos:7
MAINTAINER dev.kuro.obi@gmail.com

RUN yum -y install wget git rpmdevtools yum-utils
RUN rpmdev-setuptree
RUN wget https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.8.3.linux-amd64.tar.gz
ENV PATH $PATH:/usr/local/go/bin
RUN go get github.com/coreos/go-semver && \
    go get github.com/nightlyone/lockfile && \
    go get github.com/labstack/echo && \
    go get github.com/jessevdk/go-flags && \
    go get github.com/dgrijalva/jwt-go && \
    go get github.com/lestrrat/go-server-starter/listener && \
    go get golang.org/x/net/netutil && \
    go get github.com/monochromegane/conflag && \
    go get github.com/gansidui/priority_queue
RUN mkdir -p /root/go/src/github.com/monochromegane/gannoy
ADD . /root/go/src/github.com/monochromegane/gannoy
WORKDIR /root/go/src/github.com/monochromegane/gannoy
RUN go build -o /root/rpmbuild/SOURCES/gannoy-0.0.1 cmd/gannoy/main.go && \
    go build -o /root/rpmbuild/SOURCES/gannoy-converter-0.0.1 cmd/gannoy-converter/main.go && \
    go build -o /root/rpmbuild/SOURCES/gannoy-db-0.0.1 cmd/gannoy-db/main.go
WORKDIR /root
ADD rpmbuild/SPECS/gannoy.spec /root/rpmbuild/SPECS/gannoy.spec
ADD rpmbuild/SOURCES/gannoy-* /root/rpmbuild/SOURCES/
RUN rpmbuild -bb /root/rpmbuild/SPECS/gannoy.spec
# RUN rpm -ivh /root/rpmbuild/RPMS/x86_64/gannoy-0.0.1-1.x86_64.rpm
