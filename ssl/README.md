# To create sample keys:

```
openssl genrsa -out server.key 2048
openssl req -new -x509 -key server.key -days 3650 -nodes -out server.crt -keyout server.crt
```

## TODO

* check if ed25519 keys work
