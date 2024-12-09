cd ../
go build
chmod +x olympsis-server
if [ $? -ne 0 ]; then  
echo "Error: Failed to build new server binary."  
exit 1
fi

systemctl restart olympsis-server

rm /sbin/olympsis-server
mv olympsis-server /sbin
if [ $? -ne 0 ]; then  
echo "Error: Failed to move binary."  
exit 1
fi

echo "Update Successful"