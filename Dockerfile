#Download base image ubuntu 20.04
FROM ubuntu:20.04
LABEL version="0.1"
LABEL description="This is custom Docker Image for \
    Lava Go Test"

# Disable Prompt During Packages Installation
ARG DEBIAN_FRONTEND=noninteractive

# Update Ubuntu Software repository
RUN apt update
RUN apt install wget -y
RUN pwd
RUN wget https://go.dev/dl/go1.18.2.linux-amd64.tar.gz -O go1.18.2.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz
RUN export PATH=$PATH:/usr/local/go/bin
RUN export PATH=$PATH:$(go env GOPATH)/bin
RUN export GOPATH=$(go env GOPATH)
RUN go version
RUN curl https://get.ignite.com/cli! | bash
RUN ignite version
RUN curl https://get.starport.network/starport@v0.19.2! | bash
RUN starport version


RUN apt install -y nginx php-fpm supervisor && \
    rm -rf /var/lib/apt/lists/* && \
    apt clean