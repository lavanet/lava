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

# install ignite
RUN curl https://get.ignite.com/cli! | bash
RUN ignite version
RUN echo ":::::::::::::::::::::::::: IGNITE INSTALLED"

# Get Starport
# RUN curl https://get.starport.network/starport@v0.19.2! | bash
# RUN starport version
# RUN echo ":::::::::::::::::::::::::: Starport Installed"



# Check deps
RUN ls -l /lib/x86_64-linux-gnu
RUN /sbin/ldconfig -p | grep stdc++  

ENV LAVA=$GOPATH/lava
RUN echo ":::::::::::::::::::::::::: @@@ LAVA: " $LAVA

ADD . $GOPATH/lava
RUN echo ":::::::::::::::::::::::::: !!! pwd: " `pwd`
RUN ls -l

# RUN cd /go/lava && ignite chain serve -r -v
# RUN cd /go/lava && go test ./testutil/e2e -v
RUN chmod +x scripts/lava_node.sh scripts/lava_full.sh
RUN chmod +x -R scripts/

# VOLUME [ "/go/lava/docker/shared" ]

# CMD sh scripts/lava_node.sh
# CMD (cd $LAVA && starport chain serve -v -r 2>&1 | grep -e lava_ -e ERR_ -e STARPORT] -e !
CMD ignite chain serve -v
# ENTRYPOINT $GOPATH/lava/scripts/lava_node.sh 

# 🌍 Token faucet API
EXPOSE 4500
# 🌍 Blockchain API: 
EXPOSE 1317
# 🌍 Tendermint node: 
EXPOSE 26657

#   To build docker locally
# $ docker build . -t lava_starport
#
#   To run docker
# $ docker run -p 4500:4500 -p 1317:1317 -p 26657:26657 --name lavaIgnite lava_starport |& grep -e lava_ -e ERR_ -e STARPORT] -e !
#
#   init
# $ docker exec -it lava19 sh scripts/init.sh
