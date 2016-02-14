To use letsencrypt follow these steps:

```
git clone https://github.com/letsencrypt/letsencrypt
cd letsencrypt
sudo ./letsencrypt-auto certonly --standalone --email youremail@somewhere.com -d yourserver.com
```

Use the NGINX block in this directory. Then startup `awwkoala` with

```bash
sudo ./awwkoala -p :8001 -key /etc/letsencrypt/live/yourserver.com/privkey.pem -crt /etc/letsencrypt/live/yourserver.com/cert.pem yourserver.com
```
