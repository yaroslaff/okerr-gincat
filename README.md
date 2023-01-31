# okerr-gincat
okerr-cat implementation in Golang with Gin

## Build, install
~~~
go build
cp okerr-gincat /usr/local/bin
cp cat.html.tmpl /etc/okerr/
echo ROLE=sorry > /etc/default/okerr-cat

cp okerr-gincat.service /etc/systemd/system/
systemctl enable --now okerr-gincat
~~~

Nginx part:
~~~
server {
    listen 80;
    server_name localhost cat.okerr.com *.cat.okerr.com;

    return 301 https://$host$request_uri;

    #location / {
    #    include uwsgi_params;
    #    uwsgi_pass unix:/opt/venv/okerr-cat/run/okerr-cat.sock;
    #}
}

server {
	listen 443 ssl;
	server_name localhost cat.okerr.com *.cat.okerr.com;

	ssl_certificate     /var/lib/dehydrated/certs/cat.okerr.com/fullchain.pem;
    	ssl_certificate_key /var/lib/dehydrated/certs/cat.okerr.com/privkey.pem;
	ssl_protocols 	    TLSv1.2; # TLSv1.1 TLSv1;
	error_log  /var/log/nginx/cat-error.log;
	access_log /var/log/nginx/cat-access.log;

	# openssl dhparam -out /etc/nginx/dhparam.pem 4096
	ssl_dhparam /etc/nginx/dhparam.pem;

	ssl_ciphers 'kEECDH+ECDSA+AES128 kEECDH+ECDSA+AES256 kEECDH+AES128 kEECDH+AES256 kEDH+AES128 kEDH+AES256 DES-CBC3-SHA +SHA !aNULL !eNULL !LOW !kECDH !DSS !MD5 !RC4 !EXP !PSK !SRP !CAMELLIA !SEED';
	ssl_prefer_server_ciphers on;

	add_header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload" always;	
	
	location / {
        	proxy_pass http://localhost:8080;
    	}
}
~~~