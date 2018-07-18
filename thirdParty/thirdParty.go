package thirdParty
import (
	"server/msg"
	"fmt"
	"bytes"
	"net/http"
	"io/ioutil"
	"encoding/json"//json封装解析
)

const wx_appid = "wx92a437da81573148"
const wx_appsecret = "0d1792fcab07c7adc49f9c90f0b4d910"
//const wx_api ="https://api.weixin.qq.com/sns/jscode2session?appid=APPID&secret=SECRET&js_code=JSCODE&grant_type=authorization_code"

type WX_OPENID struct {
	SessionKey string `json:"session_key"`
	OpenId string `json:"openid"`
}

func GetOpenID(platform string,code string) string{
	str:=""
	var buf bytes.Buffer
	switch platform{
	  case msg.WX_Platform:
		buf.WriteString("https://api.weixin.qq.com/sns/jscode2session?appid="+wx_appid)
		buf.WriteString("&secret="+wx_appsecret)
		buf.WriteString("&js_code="+code)
		buf.WriteString("&grant_type=authorization_code")
		url:=buf.String()
		p_body:=getOpenIDWithArgs(url)
		wx_data := new(WX_OPENID)
		if json_err := json.Unmarshal(*p_body, wx_data); json_err == nil {
            str = wx_data.OpenId
		}
	  default:
	}
	return str
}

func getOpenIDWithArgs(url string) *[]byte{
	resp, err := http.Get(url)
    if err != nil {
        fmt.Println("error:", err)
        return nil
    }
    defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return &body
}