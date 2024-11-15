# Asynchronous BFT Library - IACS

## Description

Asynchronous fault-tolerant protocols for the following paper:

Sourav Das, Sisi Duan, Shengqi Liu, Atsuki Momose, Ling Ren and Victor Shoup. "Asynchronous Consensus without Trusted Setup or Public-Key Cryptography." CCS 2024.

Eprint version: https://eprint.iacr.org/2024/677

This repository implements three BFT protocols:

- IACS
- FIN(for evaluation only, Sisi Duan, Xin Wang and Haibin Zhang. FIN: Practical Signature-Free Asynchronous Common Subset in Constant Time. CCS 2023.)
- PACE(for evaluation only, Haibin Zhang and Sisi Duan. PACE: Fully Parallelizable Asynchronous BFT from Reproposable Byzantine Agreement. CCS 2022.)

Scripts for evaluation on Amazon EC2 are not povided in this repo.

## Configuration
Configuration is under etc/conf.json

Change "consensus" to switch between the protocols. See note.txt for details.

## Experimental environment && Dependency

last accessed:20241115

- system: ubuntu 20.04
- github.com/cbergoon/merkletree v0.2.0
- github.com/klauspost/reedsolomon v1.12.4
- github.com/vmihailenco/msgpack v4.0.4+incompatible
- google.golang.org/grpc v1.67.1
- google.golang.org/protobuf v1.35.1
- github.com/klauspost/cpuid/v2 v2.2.8 // indirect
- golang.org/x/net v0.28.0 // indirect
- golang.org/x/sys v0.24.0 // indirect
- golang.org/x/text v0.17.0 // indirect
- google.golang.org/appengine v1.6.8 // indirect
- google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
- gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect

## Installation && How to run the code

### Install dependencies

Enter the directory and run the following commands:

```
export GO111MODULE=on
go build src/main/server.go
go build src/main/client.go
```

### Start the servers

Usage: ./server [Sid]

- Example for 4 servers with 1 failure node:

Enter the directory and run the following commands:

```
./server 0
./server 1
./server 2
./server 3
```

### Start the client

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
	
	  Enter the directory and run the following commands:
	```
	./client 100 0 1 hi 
	```
	
	- Start a client with ID = 100, and send 10 write requests with the default message.
	
	  Enter the directory and run the following commands:
	```
	./client 100 0 10 
	```
	
	- Start a client with ID = 100, and send batch requests with size 10 and "hi" message, frequency is 5.
	
	  Enter the directory and run the following commands:
	
	```
	./client 100 1 10 hi 5
	```
