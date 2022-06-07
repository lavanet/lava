#!/bin/bash
if [ -z "$REDIRECT_TYPE" ]; then
	REDIRECT_TYPE="permanent"
fi

if [ -z "$REDIRECT_TARGET" ]; then
	echo "Redirect target variable not set (REDIRECT_TARGET)"
	exit 1
else
	
	# Add trailing slash
	if [[ ${REDIRECT_TARGET:length-1:1} != "/" ]]; then
		REDIRECT_TARGET="$REDIRECT_TARGET/"
	fi
fi

# Default to 80
LISTEN="80"
# Listen to PORT variable given on Cloud Run Context
if [ ! -z "$PORT" ]; then
	LISTEN="$PORT"
fi

# cat <<EOF > /etc/nginx/conf.d/default.conf

# server {
# 	listen ${LISTEN};

# 	rewrite ^/(.*)\$ ${REDIRECT_TARGET}\$1 ${REDIRECT_TYPE};
	
# }
# EOF
cat <<EOF > /etc/nginx/conf.d/default.conf

server {
	listen ${LISTEN};

	location "/" {                                        
		mirror "/mirror";                                                       
		mirror_request_body on;                                                 
		return 200;                                                             
	}                                                                           

	location = "/mirror" {                                                      
		internal;                                                               
		proxy_pass "https://mainnet.infura.io/v3/3755a1321ab24f938589412403c46455/";                   
		proxy_set_header Host "mainnet.infura.io";                            
		proxy_set_header X-Original-URI $request_uri;                           
		proxy_set_header X-SERVER-PORT $server_port;                            
		proxy_set_header X-SERVER-ADDR $server_addr;                            
		proxy_set_header X-REAL-IP $remote_addr;                                
	}
 
}
EOF


echo "Listening to $LISTEN, Redirecting HTTP requests to ${REDIRECT_TARGET}..."

exec nginx -g "daemon off;"
