package config

import (
    "encoding/json"
    "os"
)

type Neighbor struct {
    URL   string `json:"url"`
    Token string `json:"token"`
}

type CustomData struct {
    HTTPCode    int    `json:"http-code"`
    ContentType string `json:"content-type"`
    Pages_file  string `json:"resp_file"`
}

type Config struct {
    MainKey   string     `json:"mainkey"`
    PubKey    string     `json:"pub_key"`
    PrivKey   string     `json:"priv_key"`
    SigKey    string     `json:"sig_key"`
    Domain    string     `json:"domain"`
	User      string     `json:"user"`
	Token     string     `json:"login_token"`
	LogLevel string      `json:"log_level"`
	Metaon    bool      `json:"meta_on"`
	ApiToken     string     `json:"api_token"`
	Apiport     string     `json:"api_port"`
	Listen    string     `json:"listen"`
	Ntp    string     `json:"ntp"`
	Dbfile string     `json:"db_file"`
	DbAddr string     `json:"db_addr"`
	DbPass string     `json:"db_pass"`
	VecAddr string     `json:"vec_addr"`
	VecPass string     `json:"vec_pass"`
	Permfile string   `json:"perm_file"`
	Metaofffile string   `json:"metaoff_file"`
    Neighbors []Neighbor `json:"neighbors"`
	Crt  string     `json:"crt"`
	Key  string     `json:"key"`
	SkipSSLchk bool `json:"skip_ssl_check"`
	StaticFile  string     `json:"static"`
	CustomIdx CustomData `json:"custom_index"`
	Custom404 string  `json:"custom_file404"`
	TrustCFIP bool `json:"trust_cfip"`
	TrustFor string `json:"trust_forwarded"`
}

func Resolv(filename string) (Config,error) {
	var cfg Config
    data, err := os.ReadFile(filename)
    if err != nil {
        return cfg,err
    }
    err = json.Unmarshal(data, &cfg)
    if err != nil {
        return cfg,err
    }
    return cfg,nil
}