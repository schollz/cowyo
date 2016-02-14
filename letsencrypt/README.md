To use letsencrypt follow these steps:

```
git clone https://github.com/letsencrypt/letsencrypt
cd letsencrypt
sudo ./letsencrypt-auto certonly --standalone --email youremail@somewhere.com -d yourserver.com
```

And then replace the NGINX file in `../install` with the following:

```
server {
    listen      80;
    server_name ADDRESS;
    rewrite     ^   https://$server_name$request_uri? permanent;
}

server {
  # SERVER BLOCK FOR ADDRESS
  listen   443 ssl;
  ssl_protocols       TLSv1 TLSv1.1 TLSv1.2;
  ssl_certificate         /etc/letsencrypt/live/ADDRESS/cert.pem; 
  ssl_certificate_key     /etc/letsencrypt/live/ADDRESS/privkey.pem; 

	access_log /etc/nginx/logs/access-ADDRESS.log;
	error_log /etc/nginx/logs/error-ADDRESS.log info;
	root CUR_DIR;
	server_name ADDRESS;

	# Media: images, icons, video, audio, HTC
	location ~* \.(?:jpg|jpeg|gif|png|ico|cur|gz|svg|svgz|mp4|ogg|ogv|webm|htc)$ {
		expires 1M;
		access_log off;
		add_header Cache-Control "public";
	}

	# CSS and Javascript
	location ~* \.(?:css|js)$ {
		expires 1y;
		access_log off;
		add_header Cache-Control "public";
	}

	location ^~ /static  {
		try_files $uri $uri/ =404;
	}

	location ~ ^/ {
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header Host $http_host;
		proxy_set_header X-NginX-Proxy true;

		proxy_pass https://127.0.0.1:PORT;
		proxy_redirect off;

		proxy_http_version 1.1;
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection "upgrade";
	}
}
```
