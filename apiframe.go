package main

import (
	"encoding/binary"
	"errors"

	"github.com/go-restruct/restruct"
)

const startApiFrame byte = 0xFE
const escapeApiFrame byte = 0xEA
const stopApiFrame byte = 0xEF

const echoApiRequest uint8 = 0

type EchoApiRequest struct {
	Id   uint8  `struct:"uint8"`
	Echo string `struct:"string"`
}

const firmRevApiRequest uint8 = 2

type FirmRevApiRequest struct {
	Id uint8 `struct:"uint8"`
}

const nodeIdApiRequest uint8 = 4

type NodeIdApiRequest struct {
	Id uint8 `struct:"uint8"`
}

const echoApiReply uint8 = 1

type EchoApiReply struct {
	Id   uint8  `struct:"uint8"`
	Echo string `struct:"string"`
}

const firmRevApiReply uint8 = 3

type FirmRevApiReply struct {
	Id       uint8  `struct:"uint8"`
	Revision string `struct:"string"`
}

const nodeIdApiReply uint8 = 5

type NodeIdApiReply struct {
	Id     uint8  `struct:"uint8"`
	Serial uint32 `struct:"uint32"`
}

const logEventApiReply uint8 = 57

type LogEventApiReply struct {
	Id    uint8  `struct:"uint8"`
	Level uint16 `struct:"uint16"`
	From  uint32 `struct:"uint32"`
	Line  string `struct:"string"`
}

const meshmeshProtocolConnectedPath uint8 = 7

const connectedPathApiRequest uint8 = 122

type ConnectedPathApiRequest struct {
	Id       uint8  `struct:"uint8"`
	Protocol uint8  `struct:"uint8"`
	Command  uint8  `struct:"uint8"`
	Handle   uint16 `struct:"uint16"`
	Dummy    uint16 `struct:"uint16"`
	Sequence uint16 `struct:"uint16"`
	DataSize uint16 `struct:"uint16"`
	Data     []byte `struct:"[]byte,sizefrom=DataSize"`
}

type ConnectedPathApiRequest2 struct {
	Id       uint8   `struct:"uint8"`
	Protocol uint8   `struct:"uint8"`
	Command  uint8   `struct:"uint8"`
	Handle   uint16  `struct:"uint16"`
	Dummy    uint16  `struct:"uint16"`
	Sequence uint16  `struct:"uint16"`
	DataSize uint16  `struct:"uint16"`
	Port     uint16  `struct:"uint16"`
	PathLen  uint8   `struct:"uint8"`
	Path     []int32 `struct:"[]int32,sizefrom=PathLen"`
}

const connectedPathApiReply uint8 = 123

type ConnectedPathApiReply struct {
	Id      uint8  `struct:"uint8"`
	Command uint8  `struct:"uint8"`
	Handle  uint16 `struct:"uint16"`
	Data    []byte `struct:"[]byte"`
}

type ApiFrame struct {
	data    []byte
	escaped bool
}

func (frame *ApiFrame) AwaitedReply() uint8 {
	if len(frame.data) == 0 {
		return 0
	} else {
		return (frame.data[0] & 0xFE) + 1
	}
}

func (frame *ApiFrame) AssertType(wantedType uint8) bool {
	if len(frame.data) == 0 || frame.data[0] != wantedType {
		return false
	} else {
		return true
	}
}

func (frame *ApiFrame) Escape() {
	if frame.escaped {
		return
	}

	var out []byte = []byte{}
	for _, b := range frame.data {
		if b == stopApiFrame || b == startApiFrame || b == escapeApiFrame {
			out = append(out, escapeApiFrame)
		}
		out = append(out, b)
	}

	frame.data = out
	frame.escaped = true
}

func (frame *ApiFrame) Output() []byte {
	if !frame.escaped {
		frame.Escape()
	}

	var out []byte = []byte{startApiFrame}
	out = append(out, frame.data...)
	out = append(out, stopApiFrame)
	return out
}

func (frame *ApiFrame) Decode() (interface{}, error) {
	if !frame.escaped {
		frame.Escape()
	}

	switch frame.data[0] {
	case echoApiReply:
		v := EchoApiReply{Id: 0, Echo: string(frame.data[1:])}
		return v, nil
	case firmRevApiReply:
		v := FirmRevApiReply{Id: 0, Revision: string(frame.data[1:])}
		return v, nil
	case nodeIdApiReply:
		v := NodeIdApiReply{}
		restruct.Unpack(frame.data, binary.LittleEndian, &v)
		return v, nil
	case logEventApiReply:
		v := LogEventApiReply{}
		restruct.Unpack(frame.data, binary.LittleEndian, &v)
		v.Line = string(frame.data[7:])
		return v, nil
	case connectedPathApiReply:
		v := ConnectedPathApiReply{}
		restruct.Unpack(frame.data, binary.LittleEndian, &v)
		if len(frame.data) > 4 {
			v.Data = frame.data[4:]
		}
		return v, nil
	}

	return EchoApiReply{}, errors.New("unknow api frame")
}

func (frame *ApiFrame) Encode(cmd interface{}) error {
	var b []byte
	var err error
	switch v := cmd.(type) {
	case EchoApiRequest:
		v.Id = echoApiRequest
		b, err = restruct.Pack(binary.LittleEndian, &v)
	case FirmRevApiRequest:
		v.Id = firmRevApiRequest
		b, err = restruct.Pack(binary.LittleEndian, &v)
	case NodeIdApiRequest:
		v.Id = nodeIdApiRequest
		b, err = restruct.Pack(binary.LittleEndian, &v)
	case ConnectedPathApiRequest:
		v.Id = connectedPathApiRequest
		b, err = restruct.Pack(binary.LittleEndian, &v)
	case ConnectedPathApiRequest2:
		v.Id = connectedPathApiRequest
		b, err = restruct.Pack(binary.LittleEndian, &v)
	}

	if err == nil {
		if len(b) == 0 {
			err = errors.New("can't encode requested stuct")
		} else {
			frame.data = b
			frame.escaped = true
		}
	}

	return err
}

func NewApiFrame(buffer []byte, escaped bool) *ApiFrame {
	f := &ApiFrame{
		data:    buffer,
		escaped: escaped,
	}

	return f
}

func NewApiFrameFromStruct(v interface{}) (*ApiFrame, error) {
	f := &ApiFrame{}
	err := f.Encode(v)
	return f, err
}
