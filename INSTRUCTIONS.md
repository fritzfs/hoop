git clone https://github.com/fritzfs/hoop.git

cd hoop

make build-client

chmod +x setup.sh
chmod +x connect.sh

sudo ./setup.sh

sudo ./connect.sh /tmp/hoop/... 
