apt-get update
apt-get install nginx
cp cowyo.nginx /etc/nginx/sites-available/
cp cowyo.init /etc/init.d/
ln -s /etc/nginx/sites-available/cowyo.nginx /etc/nginx/sites-enabled/cowyo.nginx
service nginx reload && service nginx restart
cd ../
go build
service cowyo.init start

