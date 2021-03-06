package libol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/xtaci/kcp-go/v5"
	"net"
	"time"
)

const (
	MAXBUF = 4096
	HSIZE  = 0x04
)

func GetHeaderLen() int {
	return HSIZE
}

var MAGIC = []byte{0xff, 0xff}

func GetMagic() []byte {
	return MAGIC
}

func IsControl(data []byte) bool {
	if len(data) < 6 {
		return false
	}
	if bytes.Equal(data[:6], ZEROED[:6]) {
		return true
	}
	return false
}

type FrameMessage struct {
	control bool
	action  string
	params  string
	frame   []byte
	rawData []byte
}

func NewFrameMessage(data []byte) *FrameMessage {
	m := FrameMessage{
		control: false,
		action:  "",
		params:  "",
		rawData: data,
	}
	m.Decode()
	return &m
}

func (m *FrameMessage) Decode() bool {
	m.control = IsControl(m.rawData)
	if m.control {
		m.action = string(m.rawData[6:11])
		m.params = string(m.rawData[12:])
	} else {
		m.frame = m.rawData
	}
	return m.control
}

func (m *FrameMessage) IsControl() bool {
	return m.control
}

func (m *FrameMessage) Data() []byte {
	return m.frame
}

func (m *FrameMessage) String() string {
	return fmt.Sprintf("control: %t, rawData: %x", m.control, m.rawData[:20])
}

func (m *FrameMessage) CmdAndParams() (string, string) {
	return m.action, m.params
}

type ControlMessage struct {
	control  bool
	operator string
	action   string
	params   string
}

//operator: request is '= ', and response is  ': '
//action: login, network etc.
//body: json string.
func NewControlMessage(action string, opr string, body string) *ControlMessage {
	c := ControlMessage{
		control:  true,
		action:   action,
		params:   body,
		operator: opr,
	}

	return &c
}

func (c *ControlMessage) Encode() []byte {
	p := fmt.Sprintf("%s%s%s", c.action[:4], c.operator[:2], c.params)
	return append(ZEROED[:6], p...)
}

type Messager interface {
	Send(conn net.Conn, data []byte) (int, error)
	Receive(conn net.Conn, data []byte, max, min int) (int, error)
}

type StreamMessage struct {
	timeout time.Duration // ns for read and write deadline.
	block   kcp.BlockCrypt
}

func (s *StreamMessage) write(conn net.Conn, tmp []byte) (int, error) {
	if s.timeout != 0 {
		err := conn.SetWriteDeadline(time.Now().Add(s.timeout))
		if err != nil {
			return 0, err
		}
	}
	n, err := conn.Write(tmp)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *StreamMessage) writeFull(conn net.Conn, buf []byte) error {
	if conn == nil {
		return NewErr("connection is nil")
	}
	offset := 0
	size := len(buf)
	left := size - offset
	Log("writeFull: %s %d", conn.RemoteAddr(), size)
	Log("writeFull: %s Data %x", conn.RemoteAddr(), buf)
	for left > 0 {
		tmp := buf[offset:]
		Log("writeFull: tmp %s %d", conn.RemoteAddr(), len(tmp))
		n, err := s.write(conn, tmp)
		if err != nil {
			return err
		}
		Log("writeFull: %s snd %d, size %d", conn.RemoteAddr(), n, size)
		offset += n
		left = size - offset
	}
	return nil
}

func (s *StreamMessage) Send(conn net.Conn, data []byte) (int, error) {
	size := len(data)
	buf := make([]byte, HSIZE+size)
	copy(buf[0:2], MAGIC)
	binary.BigEndian.PutUint16(buf[2:4], uint16(size))
	if s.block != nil {
		s.block.Encrypt(data, data)
	}
	copy(buf[HSIZE:], data)
	if err := s.writeFull(conn, buf); err != nil {
		return 0, err
	}
	return size, nil
}

func (s *StreamMessage) read(conn net.Conn, tmp []byte) (int, error) {
	if s.timeout != 0 {
		err := conn.SetReadDeadline(time.Now().Add(s.timeout))
		if err != nil {
			return 0, err
		}
	}
	n, err := conn.Read(tmp)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *StreamMessage) readFull(conn net.Conn, buf []byte) error {
	if conn == nil {
		return NewErr("connection is nil")
	}
	offset := 0
	left := len(buf)
	Log("readFull: %s %d", conn.RemoteAddr(), len(buf))
	for left > 0 {
		tmp := make([]byte, left)
		n, err := s.read(conn, tmp)
		if err != nil {
			return err
		}
		copy(buf[offset:], tmp)
		offset += n
		left -= n
	}
	Log("readFull: Data %s %x", conn.RemoteAddr(), buf)
	return nil
}

func (s *StreamMessage) Receive(conn net.Conn, data []byte, max, min int) (int, error) {
	hl := GetHeaderLen()
	buf := make([]byte, hl+max)
	h := buf[:hl]
	if err := s.readFull(conn, h); err != nil {
		return 0, err
	}
	magic := GetMagic()
	if !bytes.Equal(h[0:2], magic) {
		return 0, NewErr("%s: wrong magic", conn.RemoteAddr())
	}
	size := binary.BigEndian.Uint16(h[2:4])
	if int(size) > max || int(size) < min {
		return 0, NewErr("%s: wrong size(%d)", conn.RemoteAddr(), size)
	}
	tmp := buf[hl : hl+int(size)]
	if err := s.readFull(conn, tmp); err != nil {
		return 0, err
	}
	if s.block != nil {
		s.block.Decrypt(tmp, tmp)
	}
	copy(data, tmp)
	return len(tmp), nil
}

type DataGramMessage struct {
	timeout time.Duration // ns for read and write deadline
	block   kcp.BlockCrypt
}

func (s *DataGramMessage) Send(conn net.Conn, data []byte) (int, error) {
	size := len(data)
	buf := make([]byte, HSIZE+size)
	copy(buf[0:2], MAGIC)
	binary.BigEndian.PutUint16(buf[2:4], uint16(size))
	if s.block != nil {
		s.block.Encrypt(data, data)
	}
	copy(buf[HSIZE:], data)
	Log("DataGramMessage.Send: %s %x", conn.RemoteAddr(), data)
	if s.timeout != 0 {
		err := conn.SetWriteDeadline(time.Now().Add(s.timeout))
		if err != nil {
			return 0, err
		}
	}
	if _, err := conn.Write(buf); err != nil {
		return 0, err
	}
	return size, nil
}

func (s *DataGramMessage) Receive(conn net.Conn, data []byte, max, min int) (int, error) {
	hl := GetHeaderLen()
	buf := make([]byte, hl+max)
	Debug("DataGramMessage.Receive %s %d", conn.RemoteAddr(), s.timeout)
	if s.timeout != 0 {
		err := conn.SetReadDeadline(time.Now().Add(s.timeout))
		if err != nil {
			return 0, err
		}
	}
	n, err := conn.Read(buf)
	if err != nil {
		return 0, err
	}
	Log("DataGramMessage.Receive: %s %x", conn.RemoteAddr(), buf[:n])
	if n <= hl {
		return 0, NewErr("%s: small frame", conn.RemoteAddr())
	}
	magic := GetMagic()
	if !bytes.Equal(buf[0:2], magic) {
		return 0, NewErr("%s: wrong magic", conn.RemoteAddr())
	}
	size := binary.BigEndian.Uint16(buf[2:4])
	if int(size) > max || int(size) < min {
		return 0, NewErr("%s: wrong size(%d)", conn.RemoteAddr(), size)
	}
	tmp := buf[hl : hl+int(size)]
	if s.block != nil {
		s.block.Encrypt(tmp, tmp)
	}
	copy(data, tmp)
	return len(tmp), nil
}
