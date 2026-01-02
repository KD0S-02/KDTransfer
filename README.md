# KDTransfer

A lightweight, peer-to-peer (P2P) file transfer tool written in Go, designed to explore low-level network communication, custom binary protocols, and concurrent programming.

## Overview

KDTransfer is a P2P file sharing tool I’m building to get under the hood of network protocols.

Right now, it works on a Local Area Network (LAN) using a Signaling Server I wrote to help peers find each other's addresses, so you don't have to go digging for your local IP every time you want to send a file.

*What I'm working on right now*: I’m currently adding support for QUIC so this works over the internet. This involves figuring out NAT traversal and UDP hole punching, which is a big jump from basic LAN sockets but much more powerful.


## Features
* **Peer Discovery:** Dedicated signaling server for client registration and address lookup.
* **Custom Binary Protocol:** Hand-rolled framing (Command-Length-Payload) to handle TCP stream fragmentation.
* **End-to-End Encryption (E2EE):** Optional AES-GCM encryption with keys derived via PBKDF2 from a user-provided passphrase.
* **Efficient Streaming:** Built on Go's `io` interfaces to stream files directly from disk, ensuring low memory footprints for large transfers.
* **Connection-Racing & Interface Discovery:** Automatically scans network interfaces and filters out "noise" like Docker or virtual bridges. It retrieves both local and public IPs to "race" TCP and QUIC connections simultaneously, automatically picking the fastest available path.


## How the "Connection Race" Works
Since the app will support both TCP for LAN and QUIC for the internet, it doesn't just guess which one to use. Instead, it initiates a race:
1. **TCP Attempt:** Aimed at local network speed (LAN).
2. **QUIC Attempt:** Optimized for punching through firewalls and handling packet loss (Internet).
3. **The Winner:** Whichever handshake completes first is used for the file transfer. This ensures the best performance whether you are in the same room or across the world. (the quic part is work in progress)

---

## Quick Start

```bash
# Clone the repository
git clone https://github.com/KD0S-02/KDTransfer.git
cd KDTransfer

# Build using Makefile
make build

# Or manually
go build -o kdtransfer-server ./cmd/server  
go build -o kdtransfer ./cmd/client

# Run server
./kdtransfer-server

# Run client (in another terminal)
./kdtransfer recv  # For receiving files
./kdtransfer send --file <filepath> --peer <peerID> # For sending files

./kdtransfer recv --passphrase <passphrase> # For receiving files and for E2EE (can set custom passphrase)
./kdtransfer send --file <filepath> --peer <peerID> --passphrase <passphrase>


```


update the .env to set custom tcp ports and change the signalling server address.

---

## Protocol Design

### Message Structure

Each message follows the format:

```
[1 byte Command][4 bytes Payload Length][Payload...]
```

* Payload Length: 32-bit big-endian unsigned integer
* Payload is command-specific

| Offset | Field          | Type    | Description                               |
|--------|----------------|---------|-------------------------------------------|
| 0      | Command        | 1 Byte  | Action ID (e.g., Register, Connect, Data) |
| 1-4    | Payload Length | 4 Bytes | 32-bit Big-Endian unsigned integer        |
| 5+     | Payload        | []byte  | Command-specific data or file chunks      |

---
