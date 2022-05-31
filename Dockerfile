#Download base image ubuntu 20.04
FROM golang:1.18.2
LABEL version="0.1"
LABEL description="This is custom Docker Image for \
    Lava Go Test"

# Disable Prompt During Packages Installation
ARG DEBIAN_FRONTEND=noninteractive

# Update Ubuntu Software repository
RUN apt update
RUN apt install tree -y
RUN echo `pwd`
RUN ls -l
RUN mkdir lava
ADD . lava/.
WORKDIR /go/lava
RUN apt install wget -y
RUN pwd
RUN apt install apt-utils -y
# RUN wget https://go.dev/dl/go1.18.2.linux-amd64.tar.gz -O go1.18.2.linux-amd64.tar.gz
# RUN tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz
RUN echo `pwd`
RUN ls -l

ENV PATH    ="${PATH}:/go/lava"
ENV PATH    ="${PATH}:/go/bin"
ENV PATH    ="${PATH}:/bin"
ENV PATH    ="${PATH}:/go"
# ENV PATH    ="${PATH}:/usr/local"
# ENV PATH    ="${PATH}:`pwd`"
# ENV PATH    ="${PATH}:$(go env GOPATH)/bin"
# ENV GOPATH  ="${GOPATH}:/go"
# ENV GOPATH  ="${GOPATH}:/go/bin"
# ENV GOPATH  ="${GOPATH}:/go/lava"
# ENV GOPATH  ="${GOPATH}:`pwd`"
# ENV GOPATH  ="${GOPATH}:$(go env GOPATH)"
RUN go version
RUN curl https://get.ignite.com/cli! | bash
RUN ignite version
RUN curl https://get.starport.network/starport@v0.19.2! | bash
RUN starport version
RUN pwd
RUN ls -l
RUN which go
RUN mkdir ~/go
RUN cd ~/go
RUN mkdir /bin/go
RUN mkdir /bin/go/lava
# ADD . /bin/go/lava
RUN pwd
RUN tree
RUN apt install less grep -y
RUN ls -l /root/
# RUN cd /root/go/lava && timeout 100 ignite chain serve -r -v | less
RUN ignite chain build
ENV LAVA=/bin/go/lava
# RUN cd /go/lava && starport chain serve -r -v
# RUN cd /go/lava && go test ./testutil/e2e -v
LABEL name="Lava Docker"