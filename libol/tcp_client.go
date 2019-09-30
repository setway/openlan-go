package libol

import (
    "bytes"
    "encoding/binary"
    "errors"
    "fmt"
    "net"
    "time"
)

var (
    MAGIC  = []byte {0xff,0xff}
)

const (
    CL_INIT      = 0x00
    CL_CONNECTED = 0x01
    CL_UNAUTH    = 0x02
    CL_AUTHED    = 0x03
    CL_CLOSED    = 0xff
)

const (
    HSIZE  = 0x04
)

type TcpClient struct {
    conn *net.TCPConn
    maxsize int
    minsize int
    verbose int
    onConnected func (*TcpClient) error

    //Public variable
    TxOkay uint64
    RxOkay uint64
    TxError uint64
    Droped uint64
    Status uint8
    Addr string
    NewTime int64
}

func NewTcpClient(addr string, verbose int) (this *TcpClient) {
    this = &TcpClient {
        Addr: addr,
        conn: nil,
        maxsize: 1514,
        minsize: 15,
        verbose: verbose,
        TxOkay: 0,
        RxOkay: 0,
        TxError: 0,
        Droped: 0,
        Status: CL_INIT,
        onConnected: nil,
        NewTime: time.Now().Unix(),
    }

    return 
}

func NewTcpClientFromConn(conn *net.TCPConn, verbose int) (this *TcpClient) {
    this = &TcpClient {
        Addr: conn.RemoteAddr().String(),
        conn: conn,
        maxsize: 1514,
        minsize: 15,
        verbose: verbose,
        NewTime: time.Now().Unix(),
    }

    return 
}

func (this *TcpClient) Connect() error {
    if this.conn != nil {
        return nil
    }

    Info("TcpClient.Connect %s\n", this.Addr)
    raddr, err := net.ResolveTCPAddr("tcp", this.Addr)
    if err != nil {
        return err
    }

    conn, err := net.DialTCP("tcp", nil, raddr)
    if err != nil {
        this.conn = nil
        return err
    }

    this.conn = conn
    this.Status = CL_CONNECTED

    if this.onConnected != nil {
        this.onConnected(this)
    }

    return nil
}

func (this *TcpClient) OnConnected(on func (*TcpClient) error) {
    this.onConnected = on
}

func (this *TcpClient) Close() {
    if this.conn != nil {
        Info("TcpClient.Close %s\n", this.Addr)
        this.conn.Close()
        this.conn = nil
    }
}

func (this *TcpClient) recvn(buffer []byte) error {
    offset := 0
    left := len(buffer)
    for left > 0 {
        tmp := make([]byte, left)
        n, err := this.conn.Read(tmp)
        if err != nil {
            return err
        }
        copy(buffer[offset:], tmp)
        offset += n
        left -= n 
    }
    
    if this.IsVerbose() {
        Debug("TcpClient.recvn %d\n", len(buffer))
        Debug("TcpClient.recvn Data: % x\n", buffer)
    }

    return nil
}

func (this *TcpClient) sendn(buffer []byte) error {
    offset := 0
    size := len(buffer)
    left := size - offset
    if this.IsVerbose() {
        Debug("TcpClient.sendn %d\n", size)
        Debug("TcpClient.sendn Data: % x\n", buffer)
    }

    for left > 0 {
        tmp := buffer[offset:]
        if this.IsVerbose() {
            Debug("TcpClient.sendn tmp %d\n", len(tmp))
        }
        n, err := this.conn.Write(tmp)
        if err != nil {
            return err 
        }
        offset += n
        left = size - offset
    }
    return nil
}

func (this *TcpClient) SendMsg(data []byte) error {
    if err := this.Connect(); err != nil {
        return err
    }

    buffer := make([]byte, HSIZE + len(data))
    copy(buffer[0:2], MAGIC)
    binary.BigEndian.PutUint16(buffer[2:4], uint16(len(data)))
    copy(buffer[HSIZE:], data)

    if err := this.sendn(buffer); err != nil {
        this.TxError++
        return err
    }
    
    this.TxOkay++

    return nil
}

func (this *TcpClient) RecvMsg(data []byte) (int, error) {
    if this.IsVerbose() {
        Debug("TcpClient.RecvMsg %s", this)
    }

    if !this.IsOk() {
        return -1, errors.New("Connection isn't okay!")
    }

    h := make([]byte, HSIZE)
    if err := this.recvn(h); err != nil {
        return -1, err
    }

    if !bytes.Equal(h[0:2], MAGIC) {
        return -1, errors.New("Isn't right magic header!")
    }

    size := binary.BigEndian.Uint16(h[2:4])
    if int(size) > this.maxsize || int(size) < this.minsize {
        return -1, errors.New(fmt.Sprintf("Isn't right data size(%d)!", size))
    }

    d := make([]byte, size)
    if err := this.recvn(d); err != nil {
        return -1, err
    }

    copy(data, d)
    this.RxOkay++

    return int(size), nil
}

func (this *TcpClient) IsVerbose() bool {
    return this.verbose != 0
}

func (this *TcpClient) GetMaxSize() int {
    return this.maxsize
}

func (this *TcpClient) SetMaxSize(value int) {
    this.maxsize = value
}

func (this *TcpClient) GetMinSize() int {
    return this.minsize
}

func (this *TcpClient) IsOk() bool {
    return this.conn != nil
}

func (this *TcpClient) SendReq(action string, body string) error {
    data := EncInstReq(action, body)

    if this.IsVerbose() {
        Debug("TcpClient.SendReq %d %s\n", len(data), data[6:])
    }

    if err := this.SendMsg(data); err != nil {
        return err
    }
    return nil
}

func (this *TcpClient) SendResp(action string, body string) error {
    data := EncInstResp(action, body)

    if this.IsVerbose() {
        Debug("TcpClient.SendResp %d %s\n", len(data), data[6:])
    }

    if err := this.SendMsg(data); err != nil {
        return err
    }
    return nil
}

func (this *TcpClient) UpTime() int64 {
    return time.Now().Unix() - this.NewTime
}

func (this *TcpClient) String() string {
    return this.Addr
}