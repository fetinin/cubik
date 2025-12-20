# Yeelight Cube Lite Protocol Analysis

## Overview
Analysis of UDP communication between Android app (10.215.173.1) and Yeelight Cube Lite (192.168.1.37) on port 5540.

## Packet Structure

### Type 1: Handshake/Control Packets (Clear)

#### From Phone (Type 0x04):
```
Offset  Content
------  -------
0x00    Message Type: 04 00 00 00 (Little Endian = 0x00000004)
0x04    Phone IP: d7 80 a9 0a (10.215.173.1 reversed)
0x08    Session ID: ae f5 35 2a 85 44 8b 70 (8 bytes)
0x10    Sequence/Type: Various (e.g., 05 30, 03 10, 07 40)
0x12    Unknown: 33 bf
0x14    Unknown: 00 00
0x16    Lamp ID/Ref: d7 80 a9 0a (or other values)
0x1a+   Additional data (TLV format - Type, Length, Value)
```

**Packet 3** (Initial handshake from phone):
```
04 00 00 00              - Message Type 4
d7 80 a9 0a              - Phone IP (10.215.173.1)
ae f5 35 2a 85 44 8b 70  - Session ID
05 30 33 bf 00 00        - Control bytes
15 30                    - TLV Type 0x15
01 20                    - Length (32 bytes)
94 a4 43 ab ... [32 bytes of data - likely public key/nonce]
a7 d5 30 03              - TLV Type continuation
20 81 d5 45 ... [more data]
```

#### From Lamp (Type 0x01):
```
Offset  Content
------  -------
0x00    Message Type: 01 00 00 00 (Little Endian = 0x00000001)
0x04    Lamp ID: 0c 25 96 07 (or increments: 0d, 0e...)
0x08    Session ID: ae f5 35 2a 85 44 8b 70 (same 8 bytes)
0x10    Sequence/Type: 02 10, 06 33, etc.
0x12    Unknown: 33 bf (or similar)
0x14    Unknown: 00 00
0x16    Phone IP: d7 80 a9 0a
0x1a+   Additional data (for some responses)
```

**Packet 4** (Lamp ACK):
```
01 00 00 00              - Message Type 1 (ACK/Response)
0c 25 96 07              - Lamp identifier
ae f5 35 2a 85 44 8b 70  - Session ID (echoed)
02 10 33 bf 00 00        - Control bytes
d7 80 a9 0a              - Phone IP (echoed)
```

**Packet 5** (Lamp with crypto parameters):
```
01 00 00 00              - Message Type 1
0d 25 96 07              - Lamp ID (incremented)
ae f5 35 2a 85 44 8b 70  - Session ID
06 33 33 bf 00 00        - Control bytes
d7 80 a9 0a              - Phone IP
15 30                    - TLV Type 0x15
01 10                    - Length (16 bytes)
cd 49 bb f8 4e e8 3f 12 83 47 37 99 69 f0 70 ad  - Data (nonce/IV)
30 02                    - TLV Type 0x30
10 84 f0 d5 1f 7c e4 42 95 d2 66 d0 49 1e fb d8  - Data (16 bytes)
d4 25 03 d5 de 18        - Trailer
```

### Type 2: Encrypted Data Packets

#### From Phone (Prefix 0x00d5de00):
```
00 d5 de 00              - Magic header (phone encrypted)
XX 3f 52 00              - Sequence number (XX increments: e3, e4, e5...)
[Encrypted payload]      - Variable length encrypted data
```

Example sequences seen:
- e3 3f 52 00, e4 3f 52 00, e5 3f 52 00... (incrementing)

#### From Lamp (Prefix 0x00a7d500):
```
00 a7 d5 00              - Magic header (lamp encrypted)
XX 67 13 08              - Sequence number (XX increments: 87, 88, 89...)
[Encrypted payload]      - Variable length encrypted data
```

Example sequences seen:
- 87 67 13 08, 88 67 13 08, 89 67 13 08... (incrementing)

## Communication Flow

### Phase 1: Handshake (Packets 3-7)
1. **Packet 3**: Phone → Lamp (Type 04) - Initial hello with public key/nonce
2. **Packet 4**: Lamp → Phone (Type 01) - ACK
3. **Packet 5**: Lamp → Phone (Type 01) - Response with crypto parameters (IV/nonce)
4. **Packet 6**: Phone → Lamp (Type 04) - ACK
5. **Packet 7**: Phone → Lamp (Type 04) - Additional handshake data

### Phase 2: Encrypted Communication (Packets 8+)
All subsequent packets use encrypted format with magic headers:
- Phone packets: 0x00d5de00
- Lamp packets: 0x00a7d500

Each encrypted packet has a sequence number that increments, suggesting ordered message delivery.

## Packet Length Patterns

### Phone encrypted packets (00d5de00):
- 42 bytes: Short commands/ACKs
- 50 bytes: Medium commands
- 67-73 bytes: Longer commands (possibly turn off command)
- 118 bytes: Extended data

### Lamp encrypted packets (00a7d500):
- 42 bytes: Short responses
- 56 bytes: Medium responses
- 75-88 bytes: Longer responses
- 144-179 bytes: Extended responses (possibly status data)

## Encryption Analysis

The protocol uses **encrypted communication** after the initial handshake:

1. **Key Exchange**: TLV Type 0x15 contains 32-byte data (likely ECDH public key)
2. **IV/Nonce**: TLV Type 0x30 contains 16-byte data (likely AES IV or ChaCha20 nonce)
3. **Cipher**: Likely AES-128 or ChaCha20 based on 16-byte IV
4. **Session ID**: 8-byte identifier (ae f5 35 2a 85 44 8b 70) used throughout

## Turn Off Command Hypothesis

Looking at the encrypted packets around the turn-off action, likely candidates:
- Packets 8-11 from phone (65-118 bytes encrypted data)
- These are followed by lamp responses (packets 13-14)

The turn off command is **encrypted** and cannot be read directly without:
1. The shared encryption key
2. Knowledge of the key derivation algorithm
3. The specific cipher and mode

## Protocol Summary

```
┌─────────────────────────────────────────────────────┐
│                  HANDSHAKE PHASE                     │
├─────────────────────────────────────────────────────┤
│ 1. Phone → Lamp: Type 04 (Hello + Public Key)       │
│ 2. Lamp → Phone: Type 01 (ACK)                       │
│ 3. Lamp → Phone: Type 01 (Crypto Params + IV)       │
│ 4. Phone → Lamp: Type 04 (ACK)                       │
│ 5. Phone → Lamp: Type 04 (Finalize)                  │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│              ENCRYPTED DATA PHASE                     │
├─────────────────────────────────────────────────────┤
│ Phone packets: 00 d5 de 00 + seq + encrypted data   │
│ Lamp packets:  00 a7 d5 00 + seq + encrypted data   │
│                                                       │
│ Sequence numbers increment for message ordering      │
└─────────────────────────────────────────────────────┘
```

## Key Findings

1. **Protocol Type**: Custom UDP protocol with encryption
2. **Port**: 5540 (standard Yeelight control port)
3. **Encryption**: Yes, using key exchange during handshake
4. **Message Format**: Type-Length-Value (TLV) in handshake
5. **Session Management**: 8-byte session ID tracks connection
6. **Sequence Numbers**: Both sides maintain incrementing counters
7. **Turn Off Command**: Encrypted - cannot decode without key

## Next Steps to Fully Reverse Engineer

To decrypt the actual commands:
1. Extract shared secret from the app (reverse engineering Android APK)
2. Identify the key derivation function (HKDF, PBKDF2, etc.)
3. Determine cipher (AES-GCM, ChaCha20-Poly1305, etc.)
4. Decrypt sample packets to validate

The protocol is **security-conscious** and uses proper encryption to prevent eavesdropping and replay attacks.
