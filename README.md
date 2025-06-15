# Encrypted P2P Chat with Ephemeral Session Keys

This implementation provides secure peer-to-peer messaging using ephemeral session keys established through X25519 ECDH key exchange, authenticated with Ed25519 identity keys.

## Security Features

### 🔐 **Ephemeral Session Keys**
- Each peer-to-peer connection establishes a unique session key
- Session keys are derived using X25519 Elliptic Curve Diffie-Hellman (ECDH)
- Keys expire after 1 hour and are automatically cleaned up
- Forward secrecy: compromising long-term keys doesn't compromise past sessions

### 🛡️ **Authentication**
- Each ephemeral key exchange is signed with the peer's Ed25519 identity key
- Prevents man-in-the-middle attacks during key establishment
- Timestamp validation prevents replay attacks (5-minute window)

### 🔒 **Encryption**
- Messages encrypted with ChaCha20-Poly1305 AEAD cipher
- Uses ephemeral session keys (not static identity keys)
- Each message has a unique nonce for semantic security
- Authentication tag prevents tampering

## Architecture

### Key Components

1. **SessionManager** (`protocol/ephemeral_session.go`)
   - Manages ephemeral session keys for each peer
   - Handles key establishment and expiration
   - Thread-safe session storage

2. **Enhanced Protocol** (`protocol/encrypted_chat.go`)
   - Multi-message protocol supporting key exchange and encrypted messages
   - Automatic session establishment when needed
   - Message acknowledgments and retry logic

3. **Crypto Utilities** (`cryptoutils/x25519_chacha_crypto.go`)
   - Ed25519 to X25519 key conversion
   - ECDH shared secret computation
   - ChaCha20-Poly1305 encryption/decryption

### Message Flow

```
Peer A                          Peer B
  |                               |
  |------- Key Exchange --------->|
  |                               | (verify signature)
  |<--- Key Exchange Response ----|
  |                               |
  | (both derive shared secret)   |
  |                               |
  |------ Encrypted Message ----->|
  |<---------- ACK ---------------|
```

### Session Establishment

1. **Initiator** generates ephemeral X25519 key pair
2. **Initiator** signs ephemeral public key with Ed25519 identity
3. **Initiator** sends signed key exchange message
4. **Responder** verifies signature and generates own ephemeral key pair
5. **Responder** sends signed key exchange response
6. **Both peers** compute shared secret using X25519 ECDH
7. **Session key** derived from shared secret (ready for encryption)

## Usage

### Running the Application

```bash
go run main.go
```

### Commands

- **Send messages**: Type any text and press Enter
- **Connect to peer**: `/connect <multiaddr>`
- **List peers**: `/peers`
- **Quit**: `/quit`

### Example Session

```
🔐 Starting encrypted P2P chat with ephemeral session keys...
📝 Commands:
  - Type messages to send to connected peers
  - '/connect <multiaddr>' to connect to a peer
  - '/peers' to list connected peers
  - '/quit' to exit

/connect /ip4/127.0.0.1/tcp/4001/p2p/12D3KooW...
🔄 Connecting to 12D3KooW...
✅ Connected to 12D3KooW

Hello, this is a secure message!
🔄 Establishing session with 12D3KooW...
✅ Session established with 12D3KooW
✅ Message delivered

💬 [Private] 12D3KooW: Hi back! This message is encrypted.
```

## Security Considerations

### ✅ **What's Protected**
- **Confidentiality**: Messages encrypted with ephemeral keys
- **Authenticity**: Messages authenticated with AEAD
- **Forward Secrecy**: Past messages safe if identity keys compromised
- **Replay Protection**: Message IDs prevent duplicate processing
- **MITM Protection**: Key exchange authenticated with identity signatures

### ⚠️ **Limitations**
- **Metadata**: Connection patterns, message timing, and sizes are visible
- **Identity Keys**: Ed25519 identity keys are long-term (compromise affects authentication)
- **No Deniability**: Messages are authenticated (can prove sender)
- **Storage**: No persistent storage of messages (memory only)

### 🔧 **Recommended Improvements**
- Add deniable authentication (e.g., Double Ratchet)
- Implement message persistence with encrypted storage
- Add support for group messaging
- Implement key rotation for identity keys
- Add onion routing for metadata protection

## Dependencies

- `golang.org/x/crypto/curve25519` - X25519 ECDH
- `golang.org/x/crypto/chacha20poly1305` - AEAD encryption
- `filippo.io/edwards25519` - Ed25519 curve operations
- `github.com/libp2p/go-libp2p` - P2P networking
- `github.com/google/uuid` - Message IDs
- `github.com/hashicorp/golang-lru` - Message deduplication

## Protocol Specification

### Message Types

1. **Key Exchange** (`key_exchange`)
   ```json
   {
     "type": "key_exchange",
     "payload": {
       "ephemeral_pub_key": "[32]byte",
       "signature": "[]byte",
       "timestamp": "2025-06-15T10:30:00Z",
       "peer_id": "12D3KooW..."
     }
   }
   ```

2. **Key Exchange Response** (`key_exchange_response`)
   ```json
   {
     "type": "key_exchange_response", 
     "payload": {
       "ephemeral_pub_key": "[32]byte",
       "signature": "[]byte", 
       "timestamp": "2025-06-15T10:30:01Z",
       "peer_id": "12D3KooW..."
     }
   }
   ```

3. **Encrypted Message** (`encrypted`)
   ```json
   {
     "type": "encrypted",
     "payload": {
       "from": "12D3KooW...",
       "to": "12D3KooW...",
       "payload": "[]byte", // nonce + ciphertext
       "message_id": "uuid"
     }
   }
   ```

4. **Acknowledgment** (`ack`)
   ```json
   {
     "type": "ack",
     "payload": {
       "status": "ok",
       "message_id": "uuid"
     }
   }
   ```

This implementation provides a robust foundation for secure P2P messaging with modern cryptographic practices.