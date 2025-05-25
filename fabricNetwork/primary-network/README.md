# Running the Primary Network

## Prerequisites

Before running the primary network, ensure you have installed the following prerequisites:

- docker and docker-compose
- git
- cURL
- Node.js and npm

## Network Commands

The primary network script (`primary-network.sh`) provides various commands to manage the network.

### 1. Bringing down the network

Standard practice to do this before running the network
To shut down the network and remove all containers, volumes, and artifacts:

```bash
./primary-network.sh down
```

This command:
- Stops all running containers (peers, orderers, CAs)
- Removes Docker volumes containing ledger data
- Cleans up chaincode containers and images
- Removes crypto material and artifacts

Use this command when you need to start fresh or switch between different network configurations.

### Starting the network with a channel

To bring up the network with Certificate Authorities (CAs) and CouchDB as the state database, and create a channel:

```bash
./primary-network.sh up createChannel -ca -s couchdb
```

This command:
- Starts the network with two organizations (Org1 and Org2), each with one peer
- Creates a single orderer node
- Uses Fabric CAs to generate crypto material for all components
- Uses CouchDB as the state database (supports rich queries)
- Creates a channel named "mychannel" by default
- Joins all peers to the created channel
- Sets anchor peers for proper service discovery

The `-ca` flag specifies the use of Certificate Authorities for generating identities.
The `-s couchdb` flag enables the CouchDB state database.

### Deploying chaincode

To deploy the basic asset transfer chaincode written in Go:

```bash
./primary-network.sh deployCC -ccn basic -ccp ../asset-transfer-basic/chaincode-go -ccl go
```

This command:
- Packages the chaincode from the specified path
- Installs the chaincode on all peers in the network
- Approves the chaincode definition for both organizations
- Commits the chaincode definition to the channel
- Initializes the chaincode (if an init function is required)

Parameters explained:
- `-ccn basic`: Sets the chaincode name to "basic"
- `-ccp ../asset-transfer-basic/chaincode-go`: Specifies the path to the chaincode
- `-ccl go`: Indicates the chaincode is written in Go language

After running this command, the chaincode is ready to be invoked from client applications or using the CLI.

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

## Interacting with the Network

Once the network is up and the chaincode is deployed, you can interact with it using the Fabric SDK or through the CLI for testing purposes.

### Example: Querying the chaincode name

```bash
peer lifecycle chaincode queryinstalled
```

### Example: Invoking a transaction via CLI

```bash
peer chaincode invoke -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com \
--tls \
--cafile /home/{path/to/primary-network}/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem \
--channelID mychannel \
--name basic \
--peerAddresses localhost:7051 \
--tlsRootCertFiles /home/{path/to/primary-network}/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem \
--peerAddresses localhost:9051 \
--tlsRootCertFiles /home/{path/to/primary-network}/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem \
--ctor '{"Function":"CreateAsset","Args":["{\"PatientId\":\"001\",\"DoctorId\":\"doctor456\",\"PatientName\":\"John Doe\",\"DateOfBirth\":\"1990-01-01\",\"Prescriptions\":[{\"PrescriptionId\":\"rx789\",\"MedicationName\":\"Aspirin\",\"Dosage\":\"100mg\",\"Instructions\":\"Take once daily\"}]}"]}'
```

## REST API
Navigate to `rest-api-go/` to run the rest api server

```bash
cd ../asset-transfer-basic/rest-api-go/
```