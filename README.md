# Prerequisites
Ensure you have the following installed:
- `git`
- `curl`
- `docker`
- `jq`

Installing prerequisites
```bash
# Installing git, curl and jq
sudo apt-get update && sudo apt-get upgrade
sudo apt install git curl jq
```

```bash
# Installing Docker
# Add Docker's official GPG key:
sudo apt-get install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

Installing go (v1.22.1)
```bash
cd ~
mkdir downloads
wget https://go.dev/dl/go1.22.1.linux-amd64.tar.gz
sudo tar -xvf go1.22.1.linux-amd64.tar.gz
sudo mv go /usr/local
echo -e "export GOROOT=/usr/local/go" >> ~/.bashrc
echo -e "export GOPATH=$HOME/go" >> ~/.bashrc
echo -e "export PATH=$GOPATH/bin:$GOROOT/bin:$PATH" >> ~/.bashrc
source ~/.bashrc
```

# Clone the repository
```bash
git clone https://github.com/Bsc-com-ne-23-20/fabricNetwork.git
```
# make sure you have docker desktop installed and running in your windows if not make sure this is so before proceeding further!
Navigate to the `fabricNetwork` directory:

```bash
cd fabricNetwork
```

# Bootstrap the network

### Make all scripts executable (in current and subdirectories)

```bash
find . -type f -name "*.sh" -exec chmod +x {} \;
```

### Make binary files executable and add them to path
```
cd ./bin
find . * -exec chmod +x {} \;
echo -e "\n### UmodziRx bin files\nexport PATH=\$PATH:$(pwd)" >> ~/.bashrc
source ~/.bashrc
```

### Run the network
First, bring down the network, get rid of any files from a previous run.
```bash
cd ../primary-network
./primary-network.sh down
```

You must be in the `primary-network/` directory to run the network:

```bash
./primary-network.sh up createChannel -ca -s couchdb
```

This will launch the following Docker containers for the network, one for each organization:
- Peer nodes
- Orderer nodes
- Certificate Authorities
- CouchDB databases

# Deploy Chaincode

To deploy the chaincode, use the following command:

```bash
./primary-network.sh deployCC -ccn basic -ccp ../asset-transfer-basic/chaincode-go -ccl go
```
- Chaincode may be re-deployed without bringing down the network.
- Several chaincodes may be deployed on a single channel, but each chaincode must be unique to each channel.


## Setup paths and env variables

```bash
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/../config/
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
```

### Example: Querying the chaincode name

```bash
peer lifecycle chaincode queryinstalled
```

### Example: Invoking a transaction via CLI

```bash
peer chaincode invoke -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com \
--tls \
--cafile ${PWD}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem \
--channelID mychannel \
--name basic \
--peerAddresses localhost:7051 \
--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem \
--peerAddresses localhost:9051 \
--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem \
--ctor '{"Function":"CreateAsset","Args":["{\"PatientId\":\"001\",\"DoctorId\":\"doctor456\",\"PatientName\":\"John Doe\",\"DateOfBirth\":\"1990-01-01\",\"Prescriptions\":[{\"PrescriptionId\":\"rx789\",\"MedicationName\":\"Aspirin\",\"Dosage\":\"100mg\",\"Instructions\":\"Take once daily\"}]}"]}'
```

## REST API
Navigate to `rest-api-go/` to run the rest api server

```bash
cd ../asset-transfer-basic/rest-api-go/
go run main.go
```
