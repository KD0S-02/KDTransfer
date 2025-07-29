# KDTransfer

A simple p2p file transfer application using Go to get a better understand low level network protocols.

## Overview

This protocol facilitates **peer-to-peer client coordination** via a signalling server, using a **custom TCP-based protocol**. Each client connects to the server to:

* Register and receive a unique ID
* Discover and connect to other peers
* Prepare for direct communication

---

## Quick Start

```bash
# Clone the repository
git clone https://github.com/KD0S-02/KDTransfer.git
cd KDTransfer

# Build server
go build -o kdtransfer-server ./cmd/server

# Build client  
go build -o kdtransfer-client ./cmd/client

# Run server
./kdtransfer-server

# Run client (in another terminal)
./kdtransfer-client recv  # For receiving files
./kdtransfer-client send --file <filepath> --peer <peerID>  # For sending files
```


update the .env to set custom port and change the signalling server address.

---

## Protocol Design

### Message Structure

Each message follows the format:

```
[1 byte Command][4 bytes Payload Length][Payload...]
```

* Payload Length: 32-bit big-endian unsigned integer
* Payload is command-specific

---

## Commands

| Command Name           | Code  | Direction         | Purpose                              |
|------------------------|-------|-------------------|--------------------------------------|
| `SERVER_HELLO`         | 0x01  | Client → Server   | Register and request ID              |
| `SERVER_ACK`           | 0x02  | Server → Client   | Send client ID back                  |
| `PEER_INFO_LOOKUP`     | 0x03  | Server → Client   | Request peer’s IP:Port by ID         |
| `PEER_LOOKUP_ACK`      | 0x04  | Server → Client   | Return peer’s IP:Port by ID          |
| `PEER_INFO_FORWARD`    | 0x05  | Server → Client   | Forward peer info to another client  |
| `ERROR`                | 0x06  | Server → Client   | Report protocol or logic error       |
| `BYE`                  | 0x07  | Client → Server   | Graceful disconnect                  |
| `FILE_TRANSFER_START`  | 0x08  | Peer ↔ Peer       | Initiate file transfer               |
| `FILE_TRANSFER_DATA`   | 0x09  | Peer ↔ Peer       | Send file data chunk                 |
| `FILE_TRANSFER_END`    | 0x0A  | Peer ↔ Peer       | End file transfer                    |

---

