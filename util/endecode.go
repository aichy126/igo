package util

//字符串可逆加解密

import (
	"bytes"
	"crypto/des"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
)

var e *Encryption

type Encryption struct {
	Key []byte
}

// DesEncrypt 加密
func DesEncrypt(text, key string) string {
	if text == "" {
		return ""
	}
	src := []byte(text)
	KEY := []byte(key)
	block, err := des.NewCipher(KEY)
	if err != nil {
		return ""
	}
	bs := block.BlockSize()
	src = PKCS7Padding(src, bs)
	if len(src)%bs != 0 {
		return ""
	}
	out := make([]byte, len(src))
	dst := out
	for len(src) > 0 {
		block.Encrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	return hex.EncodeToString(out)
}

// 解密
func DesDecrypt(decrypted, key string) string {
	if decrypted == "" {
		return ""
	}

	src, err := hex.DecodeString(decrypted)
	if err != nil {
		//log.Errorf("[%s]%s", output.MonitorEncryption, err.Error())
		return ""
	}
	KEY := []byte(key)

	block, err := des.NewCipher(KEY)
	if err != nil {
		//log.Errorf("[%s]%s", output.MonitorEncryption, err.Error())
		return ""
	}
	out := make([]byte, len(src))
	dst := out
	bs := block.BlockSize()
	if len(src)%bs != 0 {
		return ""
	}
	for len(src) > 0 {
		block.Decrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	out = PKCS7UnPadding(out)
	return string(out)
}

type KeySizeError int

func (k KeySizeError) Error() string {
	return "crypto/des: invalid key size " + strconv.Itoa(int(k))
}

func PKCS7UnPadding(origData []byte) []byte {
	return bytes.TrimFunc(origData,
		func(r rune) bool {
			return r == rune(0)
		})
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(ciphertext, padtext...)
}

// EmojiDecode 表情解码
func EmojiDecode(s string) string {
	//emoji表情的数据表达式
	re := regexp.MustCompile("\\[[\\\\u0-9a-zA-Z]+\\]")
	//提取emoji数据表达式
	reg := regexp.MustCompile("\\[\\\\u|]")
	src := re.FindAllString(s, -1)
	for i := 0; i < len(src); i++ {
		e := reg.ReplaceAllString(src[i], "")
		p, err := strconv.ParseInt(e, 16, 32)
		if err == nil {
			s = strings.Replace(s, src[i], string(rune(p)), -1)
		}
	}
	return s
}

// EmojiEnCode 表情转换
func EmojiEnCode(s string) string {
	ret := ""
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		if len(string(rs[i])) == 4 {
			u := `[\u` + strconv.FormatInt(int64(rs[i]), 16) + `]`
			ret += u

		} else {
			ret += string(rs[i])
		}
	}
	return ret
}
