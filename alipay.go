package alipay

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"errors"
	"fmt"
	"time"
	"bytes"
	"net/http"
	"io/ioutil"
)

const (
	AlipayGateWay 	= "https://openapi.alipay.com/gateway.do"
	Format		= "JSON"
	SignType	= "RSA"
	Version		= "1.0"
	CharSet		= "utf-8"
	SupportMethod	= map[string]bool{"alipay.trade.precreate":true}
)

//type BaseResponseAlipay struct {
//	Code 		string `json:"code"`
//	Msg		string `json:"msg"`
//	SubCode		string `json:"sub_code"`
//	SubMsg		string `json:"sub_msg"`
//}
//
//type AlipayTradePrecreate struct {
//	BaseResponseAlipay
//	OutTradeNo	string `json:"out_trade_no"`
//	QrCode		string `json:"qr_code"`
//}

type AlipayClient struct {
	AppId		string // app id
	Format		string // JSON
	CharSet		string // utf-8
	SignType	string // RSA
	Version		string // 1.0
	NotifyUrl	string // 回调url
}



func (this *AlipayClient)initBody (method string, params map[string]string) string{
	alipayParamters := make(map[string]string)
	alipayParamters["format"] = valueOfDefault(this.Format, Format)
	alipayParamters["charset"] = valueOfDefault(this.CharSet, CharSet)
	alipayParamters["sign_type"] = valueOfDefault(this.SignType, SignType)
	alipayParamters["version"] = valueOfDefault(this.Version, Version)
	alipayParamters["app_id"] = this.AppId
	alipayParamters["method"] = method
	alipayParamters["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	alipayParamters["notify_url"] = this.NotifyUrl
	alipayParamters["biz_content"] = toJson(params)

	paramtersSign := sign(alipayParamters)
	alipayParamters["sign"] = paramtersSign

	return alipayParamters
}

func (this *AlipayClient) Submit(method string, params map[string]string) (*map[string]interface{}, error) {
	if _, ok := SupportMethod[method]; !ok {
		return nil, errors.New(fmt.Sprintf("not suport method: %s", method))
	}
	reqBody := this.initBody(method, params)

	respBody, err := DoPost(AlipayGateWay, reqBody)
	if err != nil {
		return nil, err
	}

	respMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(respBody), &respMap); err != nil {
		return nil, errors.New(fmt.Sprintf("convert resp data to json fail, respdata: %s", respBody))
	}
	respData := getRespData(method, respMap)
	if !RespCheck(respData, respMap["sign"]) {
		return nil, errors.New(fmt.Sprintf("respData sign check fail"))
	}

	if respData["msg"] == "Success" {
		return &respData
	}

	return nil
}

func DoPost(url string, body string) (string, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return "", err
	}

	client := &http.Client{}

	response, err :=client.Do(req)
	if err != nil {
		return "", err
	}

	if response.StatusCode == 200 {
		body, _ := ioutil.ReadAll(response.Body)
		bodyStr := string(body)
		return bodyStr, nil
	}
	return "", errors.New(fmt.Sprintf("error request, status : %d", response.StatusCode))
}

func getRespData(method string, respBody map[string]interface{}) map[string]interface{} {
	respName := strings.Replace(method, ".", "_", -1) + "_" + "response"
	respData, ok := respBody[respName]
	if ok {
		return respData
	}
	return nil
}

func RespCheck(respData map[string]interface{}, sign string) bool {

	return true
}

// 按照支付宝规则生成sign
func sign(param interface{}) string {
	//解析为字节数组
	paramBytes, err := json.Marshal(param)
	if err != nil {
		return ""
	}

	//重组字符串
	var sign string
	oldString := string(paramBytes)

	//为保证签名前特殊字符串没有被转码，这里解码一次
	oldString = strings.Replace(oldString, `\u003c`, "<", -1)
	oldString = strings.Replace(oldString, `\u003e`, ">", -1)

	//去除特殊标点
	oldString = strings.Replace(oldString, "\"", "", -1)
	oldString = strings.Replace(oldString, "{", "", -1)
	oldString = strings.Replace(oldString, "}", "", -1)
	paramArray := strings.Split(oldString, ",")

	for _, v := range paramArray {
		detail := strings.SplitN(v, ":", 2)
		//排除sign和sign_type
		if detail[0] != "sign" && detail[0] != "sign_type" {
			//total_fee转化为2位小数
			if detail[0] == "total_fee" {
				number, _ := strconv.ParseFloat(detail[1], 32)
				detail[1] = strconv.FormatFloat(number, 'f', 2, 64)
			}
			if sign == "" {
				sign = detail[0] + "=" + detail[1]
			} else {
				sign += "&" + detail[0] + "=" + detail[1]
			}
		}
	}

	//追加密钥
	//sign += AlipayKey

	//md5加密
	m := md5.New()
	m.Write([]byte(sign))
	sign = hex.EncodeToString(m.Sum(nil))
	return sign
}

func toJson(data map[string]string) string{
	if jsonString, err := json.Marshal(data); err != nil {
		return ""
	}else {
		return jsonString
	}
}

func valueOfDefault(value string, defaultV string) string{
	if value == "" {
		return defaultV
	}
	return value
}
