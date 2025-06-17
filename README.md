# P2P

## Chat setup
```
handle message deduplication using lru Cache
handle message ACK from receiver, TODO: use distributed in memory cache 
message : {
  type: 
  payload: 
  from:
  to:
  message_id:
}
handle session manager (sender, receiver), cleanup
```


## Sending a chat
```
sending private chat : {
  protocol_id: needs to adjust this whats the correct value
  host:
  private_key: GenerateEd25519Key
  peer_id:
  text:
}
get session using peer_id --> not exist {
  establish session with peers (libp2p stream) {
    session msg {
      peer_id
      peer pub key
      private key on instance (created node)
      CreatedAt:     now,
      LastUsed:      now,
      PeerID:        peerID,
      MessageCount:  0,
      RekeySequence: 0,
      IsRekeying:    false,
    }
    create signed key for exchange message (session key exchange msg) {
      EphemeralPubKey:
      Timestamp:       
      PeerID:         
      IsRekey:         
      RekeySequence:
      Signature: (ephemeralpub, timestamp, peerID, etc..) signed with private key
    }
    handle session exchange {
      protocol msg {
        type
        payload
      }
      send the protocol msg into (stream)
      wait for the receiver to decode the msg
      completing the session
    }
  }
}
rekeying if {
  MessageCount >= MaxMessagesBeforeRekey
  time.Since(session.CreatedAt) >= MaxTimeBeforeRekey
  do the rekey {
    init new stream
    initiate rekey
    send rekey msg to the stream and wait for response
    complete rekey
  }
}
```

Even though your code creates a new stream for every message, rekeying is still important

1. Streams are just multiplexed channels over a single TCP (or QUIC) connection.
Creating a new stream does not create a new underlying network connection or cryptographic context at the transport layer.
2. The session key is used to encrypt/decrypt your message payloads, not the stream itself. Rekeying means you periodically generate a new shared secret for encrypting your messages, making it harder for attackers to compromise long-term communication even if they somehow get access to a session key.
3. Rekeying limits the amount of data encrypted with a single key.