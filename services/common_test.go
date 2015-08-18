package services

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestGetEnvOrDefaultShouldNotMatch(t *testing.T) {
	assert := assert.New(t)
	val := GetEnvOrDefault("PATH", "def")
	assert.NotEqual(val, "def", "they should not match")
}	

func TestGetEnvOrDefaultShouldMatchDefault(t *testing.T) {
	assert := assert.New(t)
	val := GetEnvOrDefault("NON_EXISTING_ENV", "def")
	assert.Equal(val, "def", "they should match")
}

func TestMD5ShouldMatch(t *testing.T) {
	assert := assert.New(t)
	h := MD5("john")
	assert.Equal(h, "527bd5b5d689e2c32ae974c6229ff785", "they should match")
}

func TestGetRandNumShouldNotMatch(t *testing.T) {
	assert := assert.New(t)
	r := GetRandNum(100)
	r2 := GetRandNum(100)
	assert.NotEqual(r, r2, "they should not match")
}

func TestGetRandString(t *testing.T) {
	assert := assert.New(t)
	r := GetRandString(32)
	r2 := GetRandString(32)
	assert.NotEqual(r, r2, "they should not match")
}

func TestStringInStringSlice(t *testing.T) {
	assert := assert.New(t)
	ts := []string{"john"}
	found := StringInStringSlice(ts, "john")
	assert.Equal(found, true, "they should match")
}

func TestStringInStringSliceShouldBeFalse(t *testing.T) {
	assert := assert.New(t)
	ts := []string{"john"}
	found := StringInStringSlice(ts, "james")
	assert.Equal(found, false, "they should match")
}

func TestConvertKeysToUnderscore(t *testing.T) {
	assert := assert.New(t)
	m := make(map[string]interface{})
	m["MyName"] = "john"
	m["myAge"] = 30
	newMap := ConvertKeysToUnderscore(m)
	assert.Equal(newMap["my_name"], "john", "they should match")
	assert.Equal(newMap["my_age"], 30, "they should match")
}

func TestDeleteKeys(t *testing.T) {
	assert := assert.New(t)
	m := make(map[string]interface{})
	m["a"] = 1
	m["b"] = 2
	newMap := DeleteKeys(m, "a", "b")
	assert.Equal(len(newMap), 0, "they should match")
}

func TestStructToJsonToMap(t *testing.T) {
	type Test struct{
		Name string 
		Age int `json:"age"`
	}
	assert := assert.New(t)
	m, err := StructToJsonToMap(Test{ Name: "John", Age: 18 })
	assert.Nil(err)
	assert.Equal(m["Name"], "John", "must match")
	assert.Equal(m["age"], 18.0, "must match")
}

func TestGetRandNum(t *testing.T) {
	assert := assert.New(t)
	genN := GenRandNum(16)
	assert.Equal(len(genN), 16, "should match")
}

func TestNewObjectPin(t *testing.T) {
	assert := assert.New(t)
	_, err := NewObjectPin("1")
	assert.Nil(err)
}

func TestStringSliceHasDuplicates(t *testing.T) {
	assert := assert.New(t)
	testData := [][]string{
		[]string{"a","b","b",},
		[]string{"a","b","c",},
	}
	test0 := StringSliceHasDuplicates(testData[0])
	test1 := StringSliceHasDuplicates(testData[1])
	assert.Equal(test0, true, "should match")
	assert.Equal(test1, false, "should match")
}

func TestRound(t *testing.T) {
	assert := assert.New(t)
	n := float64(11) / float64(2.0)
	assert.Equal(Round(n), 6.0, "should equal to 6")
}

func TestSIfNotZero(t *testing.T) {
	assert := assert.New(t)
	v := SIfNotZero(0)
	v2 := SIfNotZero(1) 
	assert.Equal(v, "", "should be empty")
	assert.Equal(v2, "s", "should be `s`")
}