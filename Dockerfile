#Download base image ubuntu 20.04
# FROM golang:1.18.2
FROM debian:11
LABEL version="0.1"
LABEL description="This is custom Docker Image for \
    Lava Go Test"
LABEL name="Lava Docker"

# Disable Prompt During Packages Installation
ARG DEBIAN_FRONTEND=noninteractive



RUN apt-get update
RUN apt-get upgrade -y
RUN apt-get install -y apt-utils wget git curl build-essential gcc libstdc++6

RUN wget -P /tmp https://dl.google.com/go/go1.18.2.linux-amd64.tar.gz
# RUN wget https://go.dev/dl/go1.18.2.linux-amd64.tar.gz -O go1.18.2.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf /tmp/go1.18.2.linux-amd64.tar.gz
RUN rm /tmp/go1.18.2.linux-amd64.tar.gz



ENV GOPATH /go
ENV PATH $GOPATH:$GOPATH/lava:$GOPATH/bin:/usr/local/go:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin"
RUN mkdir -p "$GOPATH/lava" && chmod -R 755 "$GOPATH"




WORKDIR $GOPATH/lava

RUN echo ":::::::::::::::::::::::::SSSSSSSSSSSSSSs"
# RUN strings /usr/lib64/libstdc++.so.6 | grep GLIBCXX
# RUN pwd
# RUN ls -l
# RUN tar -C /usr/local -xzf ./gobin/go1.18.2.linux-amd64.tar.gz

# RUN echo "::::::::::::::::::::::::::"
# RUN echo "::::::::::::::::::::::::::"
RUN echo ":::::::::::::::::::::::::SSSSSSSSSSSSSSs"
RUN ls -l
RUN pwd
RUN echo "::::::::::::::::::::::::::"
RUN echo "::::::::::::::::::::::::::"

# install ignite
RUN curl https://get.ignite.com/cli! | bash
RUN ignite version

RUN echo "::::::::::::::::::::::::::"


# Update Ubuntu Software repository
# RUN echo ":::::::::::::::::::::::::: pwd: " `pwd`
# RUN ls -l
# RUN apt update 
# RUN apt upgrade -y
# RUN apt install tree -y
# RUN apt install wget -y
# RUN apt install apt-utils -y
# RUN wget https://go.dev/dl/go1.18.2.linux-amd64.tar.gz -O go1.18.2.linux-amd64.tar.gz
# RUN tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz

    # # SETUP ENVIRONMENT
    # ENV LAVA=/go/lava
    # ENV PATH    ="${PATH}:/go/lava"
    # ENV PATH    ="${PATH}:/go/bin"
    # ENV PATH    ="${PATH}:/bin"
    # ENV PATH    ="${PATH}:/go"
# ENV PATH    ="${PATH}:/usr/local"
# ENV PATH    ="${PATH}:`pwd`"
# ENV PATH    ="${PATH}:$(go env GOPATH)/bin"
# ENV GOPATH  ="${GOPATH}:/go"
# ENV GOPATH  ="${GOPATH}:/go/bin"
# ENV GOPATH  ="${GOPATH}:/go/lava"
# ENV GOPATH  ="${GOPATH}:`pwd`"
# ENV GOPATH  ="${GOPATH}:$(go env GOPATH)"


# Get Ignite and Starport
# TODO: DOWNLOAD & ADD PACKAGES TO SAVE REDOWNLOADING EVERY OTHER TIME........
# RUN curl https://get.ignite.com/cli! | bash 
# RUN ignite version
RUN curl https://get.starport.network/starport@v0.19.2! | bash
RUN starport version
RUN echo ":::::::::::::::::::::::::: Starport Installed"

RUN ls -l /lib/x86_64-linux-gnu
RUN /sbin/ldconfig -p | grep stdc++  

# RUN strings /lib/x86_64-linux-gnu/libstdc++.so.6 | grep GLIBCXX


ADD . $GOPATH/lava
RUN echo ":::::::::::::::::::::::::: !!! pwd: " `pwd`
RUN ls -l

# Which GO 
# RUN go version
# RUN which go
# RUN mkdir ~/go
# RUN cd ~/go
# RUN mkdir /bin/go
# RUN mkdir /bin/go/lava
# ADD . /bin/go/lava
# RUN pwd
# RUN tree
# RUN apt install less grep -y
# RUN ls -l /root/
# # RUN cd /root/go/lava && timeout 100 ignite chain serve -r -v | less

# # Add Lava Files
# RUN mkdir /go/lava
# ADD . /go/lava/.
# WORKDIR /go/lava
# RUN echo ":::::::::::::::::::::::::: pwd: " `pwd`
# RUN ls -l


# Build with ignite to build dependencies
RUN ignite chain build
# ENV LAVA=/bin/go/lava

# RUN cd /go/lava && starport chain serve -r -v
# RUN cd /go/lava && go test ./testutil/e2e -v
RUN chmod +x lava_node.sh
RUN chmod +x -R .scripts/

# CMD sh lava_node.sh
# CMD (cd $LAVA && starport chain serve -v -r 2>&1 | grep -e lava_ -e ERR_ -e STARPORT] -e !
CMD starport chain serve -v
# CMD ignite chain serve -v
# ENTRYPOINT $GOPATH/lava/lava_node.sh 

# 🌍 Token faucet API
EXPOSE 4500
# 🌍 Blockchain API: 
EXPOSE 1317
# 🌍 Tendermint node: 
EXPOSE 26657

# To build docker locally
# $ docker build . -t lava_starport
# To run docker
# $ docker run -p 4500:4500 -p 1317:1317 -p 26657:26657 lava_starport -r |& grep -e lava_ -e ERR_ -e STARPORT] -e !