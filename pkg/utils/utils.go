package utils

import (
	"encoding/json"
	"go/types"
	"log"
	"math/rand"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

// MarshalStr map转string
func MarshalStr(m map[string]string) string {
	byte, err := json.Marshal(m)
	if err != nil {
		log.Printf(err.Error())
		return ""
	}
	return string(byte)
}

// Marshal 将interface map 转string
func Marshal(m map[string]interface{}) string {
	byte, err := json.Marshal(m)
	if err != nil {
		log.Printf(err.Error())
		return ""
	}
	return string(byte)
}

// val 从map中获取值
func Val(msg map[string]interface{}, key string) string {
	if msg == nil {
		return ""
	}
	val := msg[key]
	if val == nil {
		return ""
	}
	switch val.(type) {
	case string:
		return val.(string)
	case map[string]interface{}:
		return Marshal(val.(map[string]interface{}))
	default:
		log.Printf("utils. val error, val is %v", val)
		return ""
	}
}

// Unmarshal 将string转map
func Unmarshal(str string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		log.Printf(err.Error())
		return data
	}
	return data
}

// Map 将数据组成map对象
func Map(args ...interface{}) map[string]interface{} {
	if len(args)%2 != 0 {
		return nil
	}
	m := map[string]interface{}{}
	for i := 0; i < len(args)/2; i++ {
		m[args[2*i].(string)] = args[2*i+1]
	}
	return m
}

// Map 将string数据组成map对象
func MapStr(args ...interface{}) map[string]string {
	if len(args)%2 != 0 {
		return nil
	}
	m := make(map[string]string)
	for i := 0; i < len(args)/2; i++ {
		m[args[2*i].(string)] = args[2*i+1].(string)
	}
	return m
}

func randInt(min, max int) int {
	return min + rand.Intn(max-min)
}

// RandStr 随机数
func RandStr(vl int) string {
	bytes := make([]byte, vl)
	for i := 0; i < vl; i++ {
		bytes[i] = byte(randInt(48, 57))
	}
	return string(bytes)
}

// Recover 抓panic
func Recover(flag string) {
	_, _, vl, _ := runtime.Caller(1)
	if err := recover(); err != nil {
		log.Printf("[%s] Recover panic line => %v", flag, vl)
		log.Printf("[%s] Recover err => %v", flag, err)
		debug.PrintStack()
	}
}

func ProcessUrlstring(url string) []string {
	urls := strings.Split(url, ",")
	for i, s := range urls {
		urls[i] = strings.TrimSpace(s)
	}
	return urls
}

func GenerateNatsUrlString(url string) string {
	var result string
	urls := strings.Split(url, ",")
	length := len(urls)
	for i, s := range urls {
		result += "nats://" + strings.TrimSpace(s)
		if length-1 != i {
			result += ","
		}
	}
	return result

}
func ProcessUrlStringWithHttp(url string) []string {
	urls := strings.Split(url, ",")
	for i, s := range urls {
		urls[i] = "http://" + strings.TrimSpace(s)
	}
	return urls
}
func ProcessUrlStringWithHttps(url string) []string {
	urls := strings.Split(url, ",")
	for i, s := range urls {
		urls[i] = "https://" + strings.TrimSpace(s)
	}
	return urls
}

func InterfaceToInt(intVal interface{}) (rValue int) {
	switch intVal.(type) {
	case types.Nil:
		rValue = 0
	case int:
		rValue = intVal.(int)
	case int32:
		rValue = int(intVal.(int32))
	case int64:
		rValue = int(intVal.(int64))
	case float32:
		rValue = int(intVal.(float32))
	case float64:
		rValue = int(intVal.(float64))
	case []byte:
		mValue := string(intVal.([]byte))
		if iValue, err := strconv.ParseInt(mValue, 10, 64); err == nil {
			rValue = int(iValue)
		}
	case string:
		if ivalue, err := strconv.ParseInt(intVal.(string), 10, 64); err == nil {
			rValue = int(ivalue)
		}
	}
	return
}

func InterfaceToInt32(infVal interface{}) (rValue int32) {
	switch infVal.(type) {
	case nil:
		rValue = 0
	case int:
		rValue = int32(infVal.(int))
	case int32:
		rValue = int32(infVal.(int32))
	case int64:
		rValue = int32(infVal.(int64))
	case float32:
		rValue = int32(infVal.(float32))
	case float64:
		rValue = int32(infVal.(float64))
	case []byte:
		zValue := string(infVal.([]byte))
		if iValue, err := strconv.ParseInt(zValue, 10, 64); err == nil {
			rValue = int32(iValue)
		}
	case string:
		if iValue, err := strconv.ParseInt(infVal.(string), 10, 64); err == nil {
			rValue = int32(iValue)
		}
	}
	return
}

func InterfaceToInt64(infVal interface{}) (rValue int64) {
	switch infVal.(type) {
	case nil:
		rValue = 0
	case int:
		rValue = int64(infVal.(int))
	case int32:
		rValue = int64(infVal.(int32))
	case int64:
		rValue = int64(infVal.(int64))
	case float32:
		rValue = int64(infVal.(float32))
	case float64:
		rValue = int64(infVal.(float64))
	case []byte:
		zValue := string(infVal.([]byte))
		if iValue, err := strconv.ParseInt(zValue, 10, 64); err == nil {
			rValue = int64(iValue)
		}
	case string:
		if iValue, err := strconv.ParseInt(infVal.(string), 10, 64); err == nil {
			rValue = int64(iValue)
		}
	}
	return
}

func InterfaceToString(infVal interface{}) (rValue string) {
	switch infVal.(type) {
	case nil:
		rValue = ""
	case int:
		rValue = strconv.FormatInt(int64(infVal.(int)), 10)
	case int32:
		rValue = strconv.FormatInt(int64(infVal.(int32)), 10)
	case int64:
		rValue = strconv.FormatInt(infVal.(int64), 10)
	case float32:
		rValue = strconv.FormatInt(int64(infVal.(float32)), 10)
	case float64:
		rValue = strconv.FormatInt(int64(infVal.(float64)), 10)
	case []byte:
		rValue = string(infVal.([]byte))
	case string:
		rValue = infVal.(string)
	default:
		if bytes, err := json.Marshal(infVal); err == nil {
			rValue = string(bytes)
		}
	}
	return
}

func InterfaceToBool(infVal interface{}) (rValue bool) {
	switch infVal.(type) {
	case nil:
		rValue = false
	case bool:
		rValue = infVal.(bool)
	case int:
		rValue = infVal.(int) != 0
	case int32:
		rValue = infVal.(int32) != 0
	case int64:
		rValue = infVal.(int64) != 0
	case float32:
		rValue = int32(infVal.(float32)) != 0
	case float64:
		rValue = int32(infVal.(float64)) != 0
	case []byte:
		zValue := string(infVal.([]byte))
		if iValue, err := strconv.ParseInt(zValue, 10, 64); err == nil {
			rValue = int32(iValue) != 0
		}
	case string:
		if iValue, err := strconv.ParseInt(infVal.(string), 10, 64); err == nil {
			rValue = int32(iValue) != 0
		}
	}
	return
}

func InterfaceToJsonString(infVal interface{}) (string, bool) {
	infBytes, err := json.Marshal(infVal)
	if err != nil {
		return "", false
	}
	return string(infBytes[:]), true
}

func InterfaceToStringArray(infVale interface{}) []string {
	str := make([]string, 0)
	iArr, ok := infVale.([]string)
	if ok {
		return iArr
	} else {
		iArr, ok := infVale.([]interface{})
		if ok {
			for _, v := range iArr {
				if _, ok := v.(string); ok {
					str = append(str, v.(string))
				}
			}
		}
		return str
	}
}

func StringToKindInterface(infKind reflect.Kind, zVal string) (rValue interface{}) {
	switch infKind {
	case reflect.Bool:
		rValue, _ = strconv.ParseBool(zVal)
	case reflect.Int:
		rValue, _ = strconv.Atoi(zVal)
	case reflect.Int32:
		iValue, _ := strconv.Atoi(zVal)
		rValue = int32(iValue)
	case reflect.Int64:
		iValue, _ := strconv.Atoi(zVal)
		rValue = int64(iValue)
	case reflect.Float32:
		fValue, _ := strconv.ParseFloat(zVal, 64)
		rValue = int64(fValue)
	case reflect.Float64:
		fValue, _ := strconv.ParseFloat(zVal, 64)
		rValue = int64(fValue)
	case reflect.String:
		rValue = zVal
	case reflect.Slice:
		var vInf []interface{}
		json.Unmarshal([]byte(zVal), &vInf)
		rValue = vInf
	}
	return
}
