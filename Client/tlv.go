package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Tag represents the type of the TLV tag (Tag-Length-Value)
type Tag byte

// Enum for TLV tags
const (
	HelloRequest     Tag = 0
	HelloResponse    Tag = 100
	UUIDClient       Tag = 1
	UUIDPartie       Tag = 2
	Signature        Tag = 3
	String           Tag = 11
	Int              Tag = 12
	ByteData         Tag = 13
	GameRequest      Tag = 30
	GameResponse     Tag = 130
	BoardRequest     Tag = 50
	BoardResponse    Tag = 150
	ActionRequest    Tag = 40
	ActionResponse   Tag = 140
	lobby                = 177
	LobbyRequest         = 169
	JoinLobbyRequest     = 178
	lobbyResponse        = 170
)

// Error for insufficient data
var ErrInsufficientData = errors.New("insufficient data")

// EncodeTLV encodes a message in TLV (Tag-Length-Value) format
func EncodeTLV(tag Tag, value []byte) ([]byte, error) {
	// Calculate the length of the value
	length := uint16(len(value))

	// Create a buffer for TLV encoding
	buf := new(bytes.Buffer)

	// Write the tag (1 byte)
	err := binary.Write(buf, binary.BigEndian, byte(tag))
	if err != nil {
		return nil, err
	}

	// Write the length (2 bytes)
	err = binary.Write(buf, binary.BigEndian, length)
	if err != nil {
		return nil, err
	}

	// Write the value
	_, err = buf.Write(value)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// SafeDecodeTLV is a safe function to decode TLV messages
func SafeDecodeTLV(data []byte) (Tag, []byte, int, error) {
	// Check if we have at least 3 bytes for the tag and length
	if len(data) < 3 {
		return 0, nil, 0, ErrInsufficientData
	}

	// Decode the tag and length
	tag := Tag(data[0])
	length := int(data[1])<<8 | int(data[2])

	// Check if we have enough data for the full message
	if len(data) < 3+length {
		return 0, nil, 0, ErrInsufficientData
	}

	// Extract the value
	value := data[3 : 3+length]

	return tag, value, 3 + length, nil
}

// DecodeTLV decodes a TLV (Tag-Length-Value) message
func DecodeTLV(data []byte) (Tag, []byte, error) {
	// Check that data contains at least a tag and length
	if len(data) < 3 {
		return 0, nil, errors.New("TLV data too short")
	}

	tag := Tag(data[0])
	length := binary.BigEndian.Uint16(data[1:3])

	if len(data) < int(3+length) {
		return 0, nil, errors.New("incorrect length for TLV data")
	}

	// Read the value (length bytes)
	value := data[3 : 3+length]

	return tag, value, nil
}

// GetTagName returns a string representation of the tag
func GetTagName(tag Tag) string {
	switch tag {
	case HelloRequest:
		return "HelloRequest"
	case HelloResponse:
		return "HelloResponse"
	case UUIDClient:
		return "UUIDClient"
	case UUIDPartie:
		return "UUIDPartie"
	case Signature:
		return "Signature"
	case String:
		return "String"
	case Int:
		return "Int"
	case ByteData:
		return "Byte"
	case GameRequest:
		return "GameRequest"
	case GameResponse:
		return "GameResponse"
	case ActionRequest:
		return "ActionRequest"
	case ActionResponse:
		return "ActionResponse"
	default:
		return fmt.Sprintf("Unknown(%d)", tag)
	}
}
