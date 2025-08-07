# P2P Object Storage Network

A peer-to-peer, content-addressed object storage network built in Go. This project demonstrates building a decentralized storage system from the ground up using **libp2p** for networking and **gRPC** for the API.

---

## ✨ Features

- **Decentralized Network:** No central server. Peers connect directly to each other to find and share data.  
- **Content-Addressing:** Files are identified by a unique hash of their content (CID), making the data tamper-proof and automatically deduplicated.  
- **Peer-to-Peer Discovery:** Uses a Kademlia Distributed Hash Table (DHT) for efficient peer and content discovery.  
- **gRPC API:** A modern, high-performance API for interacting with a storage node.  
- **Command-Line Interface:** A user-friendly CLI to add and retrieve files from the network.  
- **Containerized:** Includes a multi-stage Dockerfile for building a small, secure, and portable server image.

---

## 🏗️ Architecture

The system consists of a server node that participates in the P2P network and a CLI client that interacts with the server's API.

```
+-------------------------------------------------------------+
|                     User / Client                           |
|                  (p2p-storage-cli)                          |
+-----------------------+-------------------------------------+
                        | (gRPC API on localhost:50051)
+-----------------------v-------------------------------------+
| Node (Server)         | APPLICATION LAYER                   |
|                       |                                     |
|  +-----------------+  |  +-------------------------------+  |
|  |   gRPC Server   |  |  |      Storage Logic            |  |
|  | (API)           +<---->+ (Chunking, Manifests)       |  |
|  +-----------------+  |  +-------------------------------+  |
+-----------------------+-------------------------------------+
                        | (libp2p function calls)
+-----------------------v-------------------------------------+
|                       | NETWORK LAYER (go-libp2p)           |
|                       |                                     |
|  +-----------------+  |  +-------------------------------+  |
|  |      Host       |  |  |       DHT (Kademlia)          |  |
|  | (PeerID, Muxer) +<---->+ (Peer & Content Routing)      |  |
|  +-----------------+  |  +-------------------------------+  |
+-----------------------+-------------------------------------+
                        | (Read/Write Blocks)
+-----------------------v-------------------------------------+
|                       | STORAGE LAYER                       |
|                       |                                     |
|  +-------------------------------------------------------+  |
|  |    BlockStore (BadgerDB)                              |  |
|  +-------------------------------------------------------+  |
+-------------------------------------------------------------+
```

---

## 🚀 Getting Started

### Prerequisites

- Go (version 1.23 or later)  
- Protocol Buffers Compiler (`protoc`)  
- Docker (for containerization)

### Installation

Clone the repository:

```bash
git clone https://github.com/your-username/p2p-storage.git
cd p2p-storage
```

Install Go dependencies for gRPC and Protobuf:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Generate code from `.proto` files:

```bash
protoc --go_out=. --go_opt=paths=source_relative   --go-grpc_out=. --go-grpc_opt=paths=source_relative   api/v1/storage.proto
```

Download project dependencies:

```bash
go mod tidy
```

---

## 💻 Usage

### 1. Run the Server Node

The server is the core of the system. It joins the P2P network, stores data, and listens for API calls from the CLI.

```bash
go run ./cmd/server
```

The server will start and print its Peer ID and listen addresses. It will also start the gRPC API server on port `50051`.

### 2. Use the CLI

The CLI is your tool for interacting with your running server node.

#### Add a File

This command will take a local file, send it to your server node, which then chunks, stores, and announces it to the network.

```bash
# Create a test file
echo "Hello from the decentralized web!" > my-file.txt

# Use the CLI to add the file
go run ./cmd/cli add my-file.txt
```

The command will output the unique Root CID for your file. Copy this CID.

#### Get a File

This command retrieves a file from the network using its Root CID and saves it locally.

```bash
# Use the CID from the 'add' command
go run ./cmd/cli get <your-root-cid> downloaded-file.txt
```

A new file, `downloaded-file.txt`, will be created with the original content.

---

## 🐳 Docker

You can also build and run the server node as a lightweight Docker container.

### Build the image:

```bash
docker build -t p2p-storage-server .
```

### Run the container:

```bash
docker run -p 50051:50051 --name my-p2p-node p2p-storage-server
```

This will start the node and expose the gRPC port, allowing your local CLI to connect to the node running inside the container.

---

## 🔮 Future Work

This project serves as a solid foundation. Future enhancements could include:

- **Data Redundancy:** Implement a replication strategy to ensure files remain available even if the original seeder node goes offline.  
- **Performance Caching:** Add an in-memory LRU cache to speed up access to frequently requested data.  
- **Encrypted Storage:** Add a layer to encrypt all data chunks before they are stored on disk or sent over the network.