package point

import (
    "flag"
    "strings"
    "fmt"
)

type Config struct {
    Addr string
    Verbose int
    Ifmtu int
    Name string
    Password string
    Auth string
}

func RightAddr(listen *string, port int) {
    values := strings.Split(*listen, ":")
    if len(values) == 1 {
        *listen = fmt.Sprintf("%s:%d", values[0], port)
    }
}

func NewConfig() (this *Config) {
    this = &Config {}

    flag.StringVar(&this.Addr, "addr", "openlan.net:10002",  "the server connect to")
    flag.IntVar(&this.Verbose, "verbose", 0x00, "open verbose")
    flag.IntVar(&this.Ifmtu, "ifmtu", 1514, "the interface MTU include ethernet")
    flag.StringVar(&this.Auth, "auth", "openlan:password",  "the auth login to")
    
    flag.Parse()
    
    values := strings.Split(this.Auth, ":")
    this.Name = values[0] 
    if (len(values) > 1) {
        this.Password = values[1]
    }

    RightAddr(&this.Addr, 10002)
    
    return
}