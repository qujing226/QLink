package connect

import (
	"encoding/binary"
	"fmt"
	didproto "github.com/qujing226/QLink/spec/gen"
	"google.golang.org/protobuf/proto"
	"net"
)

func WritePacket(conn net.Conn, pkt *didproto.Packet) error {
	data, err := proto.Marshal(pkt)
	if err != nil {
		return err
	}
	length := uint32(len(data))
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf, length)
	copy(buf[4:], data)
	_, err = conn.Write(buf)
	return err
}

func ReadPacket(conn net.Conn) (*didproto.Packet, error) {
	var length uint32
	err := binary.Read(conn, binary.BigEndian, &length)
	if err != nil {
		return nil, err
	}
	data := make([]byte, length)
	_, err = conn.Read(data)
	pkt := &didproto.Packet{}
	if err = proto.Unmarshal(data, pkt); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	return pkt, nil
}
