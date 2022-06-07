FROM nginx:1.15-alpine

COPY proxy.sh /usr/local/bin/

RUN apk add --update bash \
	&& rm -rf /var/cache/apk/* \
	&& chmod +x /usr/local/bin/proxy.sh

EXPOSE 80

CMD ["proxy.sh"]
