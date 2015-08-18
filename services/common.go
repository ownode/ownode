package services

import (
	"os"
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"time"
	"errors"
	"strconv"
	"reflect"
	"encoding/json"
	"strings"
	cryptrand "crypto/rand"
	b64 "encoding/base64"
	"golang.org/x/crypto/bcrypt"
	"github.com/fatih/camelcase"
	"github.com/DigiExam/luhn"
	"math"
)

// get an environment variable value.
// if not set, return a default value
func GetEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// return the md5 hash of a string
func MD5(str string) string {
	h := md5.New()
	io.WriteString(h, str)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// returns a random number between 0 and n
func GetRandNum(n int) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Intn(n)
}

// convert int to string
func IntToStr(n int) string {
	return strconv.Itoa(n)
}

// convert string to int
func StrToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// return's a randon string
func GetRandString(n int) string {
    const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
    var bytes = make([]byte, n)
    cryptrand.Read(bytes)
    for i, b := range bytes {
        bytes[i] = alphanum[b % byte(len(alphanum))]
    }
    return string(bytes)
}

// returns randon number of a specific length
func GenRandNum(n int) string {
    const alphanum = "0123456789"
    var bytes = make([]byte, n)
    cryptrand.Read(bytes)
    for i, b := range bytes {
        bytes[i] = alphanum[b % byte(len(alphanum))]
    }
    return string(bytes)
}

// return random numbers between a range
func GetRandNumRange(min, max int) int {
    r := rand.New(rand.NewSource(time.Now().UnixNano()))
    return r.Intn(max - min) + min
}

// check if a string starts with a sub string
func StringStartsWith(str, snippet string) bool {
	return strings.HasPrefix(str, snippet)
}

// check if a string ends with a sub string
func StringEndsWith(str, snippet string) bool {
	return strings.HasSuffix(str, snippet)
}

// split string by whitespace
func StringSplitBySpace(str string) []string {
	return strings.Fields(str)
}

// decode base64 string
func DecodeB64(str string) string {
	sDec, _ := b64.StdEncoding.DecodeString(str)
	return string(sDec)
}

// split a string by a delimeter
func StringSplit(str, delimeter string) []string {
	return strings.Split(str, delimeter)
}

// fmt.Println
func Println(args... interface{}) {
	fmt.Println(args...)
}

// check if substring exists in string
func StringSubStrExist(str, substr string) bool {
	return strings.Index(str, substr) != -1
}

// find a string in a string slice
func StringInStringSlice(sl []string, str string) bool {
	for _, s := range sl {
		if strings.ToLower(s) == strings.ToLower(str) {
			return true
		}
	}
	return false
}

// bcrypt hash a string/password
func Bcrypt(str string, cost int) (string, error) {
	hashedStr, err := bcrypt.GenerateFromPassword([]byte(str), cost)
	if err != nil {
		return "", err
	} 
	return string(hashedStr), nil
}

// compare bcrypted password and an unhashed passoword
func BcryptCompare(hashedPass, pass string) bool {
	if bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(pass)) == nil {
		return true
	}
	return false
}

// convert keys of a map to underscore
func ConvertKeysToUnderscore(m map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range m {
		splitted := camelcase.Split(k)
		newMap[strings.ToLower(strings.Join(splitted, "_"))] = v
	}
	return newMap
}

// delete keys from a map
func DeleteKeys(m map[string]interface{}, keys... string) map[string]interface{} {
	for _, k := range keys {
		delete(m, k)
	}
	return m
}

// convert a struct to json and then to a map
func StructToJsonToMap(obj interface{}) (map[string]interface{}, error) {
	var d map[string]interface{}
	jsonData, err := json.Marshal(obj)
	if err == nil {
		if err := json.Unmarshal(jsonData, &d); err != nil {
	        return d, err
	    }
	    return d, nil
	}
	return d, err
}

// convert a slice of struct to json and then to a slice of maps
func StructToJsonToSlice(obj interface{}) ([]map[string]interface{}, error) {
	var d []map[string]interface{}
	typ := reflect.TypeOf(obj)
	if typ.Kind() == reflect.Slice {
		jsonData, err := json.Marshal(obj)
		if err == nil {
			if err := json.Unmarshal(jsonData, &d); err != nil {
		        return d, err
		    }
		    return d, nil
		}
		return d, err
	} 
	return d, errors.New("invalid object type passed")
}

// create new object pin
func NewObjectPin(countryCode string) (string, error) {

	countryCodeLen := len(countryCode)
	if countryCodeLen == 0 {
		return "", errors.New("provide country code")
	}

	// pad country code if less than 4
	if countryCodeLen < 4 {
		missingLength := 4 - countryCodeLen
		countryCode = strings.Repeat("0", missingLength) + countryCode
	}

	pin := countryCode + GenRandNum(11)
	return luhn.Append(pin)
}

// checks if a slice has a duplicate string
func StringSliceHasDuplicates(data []string) bool {
	seenMap := make(map[string]struct{})
	for _, d := range data {
		if _, seen := seenMap[d]; !seen {
			seenMap[d] = struct{}{}
			continue
		}
		return true
	}
	return false
}

// convert unix timestamp to time
func UnixToTime(t int64) time.Time {
	return time.Unix(t, 0)
}

// round a number
func Round(f float64) float64 {
    return math.Floor(f + .5)
}

// returns "s" if l not zero. useful for pluralizing words
func SIfNotZero(l int) string {
	if l == 0 {
		return ""
	} 
	return "s"
}


