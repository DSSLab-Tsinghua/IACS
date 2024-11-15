# Asynchronous BFT Library - ACS

## Description
Implementation of asynchronous BFT protocols - ACS from a modular software framework. 
```bash
@misc{cryptoeprint:2024/677,
      author = {Sourav Das and Sisi Duan and Shengqi Liu and Atsuki Momose and Ling Ren and Victor Shoup},
      title = {Asynchronous Consensus without Trusted Setup or Public-Key Cryptography},
      howpublished = {Cryptology {ePrint} Archive, Paper 2024/677},
      year = {2024},
      doi = {10.1145/3658644.3670327},
      url = {https://eprint.iacr.org/2024/677}
}
```

This repository implements three protocols:

- ACS
- FIN
- PACE 


Different RBC modules are implemented under src/broadcast, and different ABA modules are impleented under src/aba. See the README files under each folder for details.

Scripts for evaluation on Amazon EC2 are not povided in this repo. 

## Configuration
Configuration is under etc/conf.json

Change "consensus" to switch between the protocols. See note.txt for details.

## Installation && How to run the code

#### 1.  Configuration environment: 
Use gomodule mode
```
/usr/local/go/bin/go env -w GO111MODULE=on
```

#### 2.  Go build:
Use the following command under the acs project directory

```
sudo rm -rf /var/log
/usr/local/go/bin/go build src/main/server.go
/usr/local/go/bin/go build src/main/client.go
```

#### 3.  Start the servers:
Usage: ./server [Sid]

- Example for 4 servers with 1 failure node:

```
./server 0
./server 1
./server 2
./server 3
```

#### 4.  Start the client:
Usage: ./client [Cid] [TypeOfRequest] [BatchSize] [Message] [Frequency]

```
Optional arguments:
	Cid					Client ID. Default is '0'.
	TypeOfRequest		Type of request. Default is 1.
						0 for write request, 1 for write batch request. 
	BatchSize         	Batch Size of the request. Default is 1.
	Message				The sending message. The size of the default message
						is 250 byte.
	Frequency         	Frequency for message sending. Default is 1.
```

- Eamples:
	- Start a client with ID = 100, and send one write request with content "hi".
	```
	./client 100 0 1 hi 
	```

	- Start a client with ID = 100, and send 10 write requests with the default message.
	```
	./client 100 0 10 
	```
	
	- Start a client with ID = 100, and send batch requests with size 10 and "hi" message, frequency is 5.
	```
	./client 100 1 10 hi 5
	```
