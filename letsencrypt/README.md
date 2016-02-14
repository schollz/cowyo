To use letsencrypt follow these steps:

```
git clone https://github.com/letsencrypt/letsencrypt
cd letsencrypt
sudo ./letsencrypt-auto certonly --standalone --email youremail@somewhere.com -d yourserver.com
```
