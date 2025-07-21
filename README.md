# KDTransfer

A simple p2p file transfer application using Go. 

# Custom Protocol

## Overview

This protocol facilitates **peer-to-peer client coordination** via a central server, using a **binary TCP-based protocol**. Each client connects to the server to:

* Register and receive a unique ID
* Discover and connect to other peers
* Prepare for direct communication

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

| Command Name        | Code | Direction       | Purpose                        |
| --------------------| ---- | --------------- | ------------------------------ |
| `SERVER_HELLO`      | 0x01 | Client → Server | Register and request ID        |
| `SERVER_ACK`        | 0x02 | Server → Client | Send client ID back            |
| `PEER_INFO_LOOKUP`  | 0x03 | Server → Client | Request peer’s IP\:Port by ID   |
| `PEER_INFO`         | 0x04 | Server → Client | Return peer’s IP\:Port by ID   |
| `ERROR`             | 0x05 | Server → Client | Report protocol or logic error |
| `BYE`               | 0x06 | Client → Server | Graceful disconnect            |

---

