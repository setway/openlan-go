package config

import (
	"flag"
	"fmt"
	"github.com/danieldin95/openlan-go/libol"
	"path/filepath"
)

type Bridge struct {
	Name     string `json:"name"`
	IfMtu    int    `json:"mtu"`
	Address  string `json:"address,omitempty" yaml:"address,omitempty"`
	Provider string `json:"provider"`
}

type IpSubnet struct {
	Start   string `json:"start"`
	End     string `json:"end"`
	Netmask string `json:"netmask"`
}

type PrefixRoute struct {
	Prefix  string `json:"prefix"`
	NextHop string `json:"nexthop"`
}

type Password struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Network struct {
	Alias    string        `json:"-"`
	Name     string        `json:"name" yaml:"name"`
	Bridge   Bridge        `json:"bridge" yaml:"bridge"`
	Links    []*Point      `json:"links" yaml:"links"`
	Routes   []PrefixRoute `json:"routes"`
	Subnet   IpSubnet      `json:"subnet"`
	Password []Password    `json:"password"`
}

func (n *Network) Right() {
	if n.Bridge.Name == "" {
		n.Bridge.Name = "br-" + n.Name
	}
	if n.Bridge.Provider == "" {
		n.Bridge.Provider = "linux"
	}
	if n.Bridge.IfMtu == 0 {
		n.Bridge.IfMtu = 1518
	}
}

type Cert struct {
	Dir     string `json:"dir"`
	CrtFile string `json:"crt" yaml:"crt"`
	KeyFile string `json:"key" yaml:"key"`
}

type FlowRules struct {
	Table    string `json:"table"`
	Chain    string `json:"chain"`
	Input    string `json:"input"`
	Source   string `json:"source"`
	ToSource string `json:"to-source"`
	Dest     string `json:"destination"`
	ToDest   string `json:"to-destination"`
	Output   string `json:"output"`
	Comment  string `json:"comment"`
	Jump     string `json:"jump"` // SNAT/RETURN/MASQUERADE
}

type Switch struct {
	Alias     string      `json:"alias"`
	Protocol  string      `json:"protocol"` // tcp/tls/kcp.
	Listen    string      `json:"listen"`
	Timeout   int         `json:"timeout"`
	Http      *Http       `json:"http,omitempty" yaml:"http,omitempty"`
	Log       Log         `json:"log" yaml:"log"`
	Cert      Cert        `json:"cert"`
	Crypt     *Crypt      `json:"crypt"`
	Network   []*Network  `json:"network"`
	FireWall  []FlowRules `json:"firewall"`
	ConfDir   string      `json:"-" yaml:"-"`
	TokenFile string      `json:"-" yaml:"-"`
	SaveFile  string      `json:"-" yaml:"-"`
}

var sd = Switch{
	Alias:   "",
	Timeout: 5 * 60,
	Log: Log{
		File:    "./openlan-switch.log",
		Verbose: libol.INFO,
	},
	Http: &Http{
		Listen: "0.0.0.0:10000",
	},
	Listen: "0.0.0.0:10002",
}

func NewSwitch() (c Switch) {
	flag.IntVar(&c.Log.Verbose, "log:level", sd.Log.Verbose, "Configure log level")
	flag.StringVar(&c.ConfDir, "conf:dir", sd.ConfDir, "Configure virtual switch directory")
	flag.Parse()

	c.SaveFile = fmt.Sprintf("%s/switch.json", c.ConfDir)
	if err := c.Load(); err != nil {
		libol.Error("NewSwitch.load %s", err)
	}
	c.Default()
	libol.Init(c.Log.File, c.Log.Verbose)
	libol.Debug("NewSwitch %v", c)
	return c
}

func (c *Switch) Right() {
	if c.Alias == "" {
		c.Alias = GetAlias()
	}
	RightAddr(&c.Listen, 10002)
	if c.Http != nil {
		RightAddr(&c.Http.Listen, 10000)
	}
	c.TokenFile = fmt.Sprintf("%s/token", c.ConfDir)
	c.SaveFile = fmt.Sprintf("%s/switch.json", c.ConfDir)
	if c.Cert.Dir != "" {
		c.Cert.CrtFile = fmt.Sprintf("%s/crt.pem", c.Cert.Dir)
		c.Cert.KeyFile = fmt.Sprintf("%s/private.key", c.Cert.Dir)
	}
}

func (c *Switch) Default() {
	c.Right()
	if c.Network == nil {
		c.Network = make([]*Network, 0, 32)
	}
	if c.Timeout == 0 {
		c.Timeout = sd.Timeout
	}
	if c.Crypt != nil {
		c.Crypt.Default()
	}
	files, err := filepath.Glob(c.ConfDir + "/network/*.json")
	if err != nil {
		libol.Error("Switch.Default %s", err)
	}
	for _, k := range files {
		n := &Network{
			Alias: c.Alias,
		}
		if err := libol.UnmarshalLoad(n, k); err != nil {
			libol.Error("Switch.Default %s", err)
			continue
		}
		c.Network = append(c.Network, n)
	}
	for _, n := range c.Network {
		for _, link := range n.Links {
			link.Default()
		}
		n.Right()
		n.Alias = c.Alias
	}
}

func (c *Switch) Load() error {
	return libol.UnmarshalLoad(c, c.SaveFile)
}

func init() {
	sd.Right()
}
