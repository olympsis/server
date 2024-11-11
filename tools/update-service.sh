go build
chmod +x olympsis-server

systemctl stop olympsis-server

rm /sbin/olympsis-server
mv olympsis-server/sbin

systemctl deamond-reload
systemctl enable olympsis-server
systemctl start olympsis-server