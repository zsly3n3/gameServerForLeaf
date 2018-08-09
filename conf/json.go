package conf

import (
	"encoding/json"
	"github.com/name5566/leaf/log"
	"io/ioutil"
)

var Server struct {
	LogLevel    string
	LogPath     string
	WSAddr      string
	CertFile    string
	KeyFile     string
	TCPAddr     string
	MaxConnNum  int
	ConsolePort int
	ProfilePath string
	DB_Name string
	DB_IP string
	DB_UserName string
	DB_Pwd string
	RemoteHttpServer string
	LocalHttpServer string
}

func init() {
	file_str:="conf/server_d.json"
	if IsRelease{
		file_str="conf/server_r.json"
	}
	data, err := ioutil.ReadFile(file_str)
	if err != nil {
		log.Fatal("%v", err)
	}
	err = json.Unmarshal(data, &Server)
	if err != nil {
		log.Fatal("%v", err)
	}
}
