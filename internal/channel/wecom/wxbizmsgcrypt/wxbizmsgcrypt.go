// Package wxbizmsgcrypt 企业微信官方加解密实现（来自 sbzhu/weworkapi_golang，MIT）
package wxbizmsgcrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"math/rand"
	"sort"
)

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const (
	ValidateSignatureError = -40001
	ParseXmlError          = -40002
	ComputeSignatureError  = -40003
	IllegalAesKey          = -40004
	ValidateCorpidError    = -40005
	EncryptAESError        = -40006
	DecryptAESError        = -40007
	IllegalBuffer          = -40008
	EncodeBase64Error      = -40009
	DecodeBase64Error      = -40010
	GenXmlError            = -40010
)

type ProtocolType int

const XmlType ProtocolType = 1

type CryptError struct {
	ErrCode int
	ErrMsg  string
}

func (e *CryptError) Error() string {
	return e.ErrMsg
}

func NewCryptError(errCode int, errMsg string) *CryptError {
	return &CryptError{ErrCode: errCode, ErrMsg: errMsg}
}

type WXBizMsg4Recv struct {
	Tousername string `xml:"ToUserName"`
	Encrypt    string `xml:"Encrypt"`
	Agentid    string `xml:"AgentID"`
}

type CDATA struct {
	Value string `xml:",cdata"`
}

type WXBizMsg4Send struct {
	XMLName   xml.Name `xml:"xml"`
	Encrypt   CDATA    `xml:"Encrypt"`
	Signature CDATA    `xml:"MsgSignature"`
	Timestamp string   `xml:"TimeStamp"`
	Nonce     CDATA    `xml:"Nonce"`
}

func NewWXBizMsg4Send(encrypt, signature, timestamp, nonce string) *WXBizMsg4Send {
	return &WXBizMsg4Send{
		Encrypt:   CDATA{Value: encrypt},
		Signature: CDATA{Value: signature},
		Timestamp: timestamp,
		Nonce:     CDATA{Value: nonce},
	}
}

type ProtocolProcessor interface {
	parse(srcData []byte) (*WXBizMsg4Recv, *CryptError)
	serialize(msgSend *WXBizMsg4Send) ([]byte, *CryptError)
}

type WXBizMsgCrypt struct {
	token              string
	encodingAESKey     string
	receiverID         string
	protocolProcessor  ProtocolProcessor
}

type XmlProcessor struct{}

func (XmlProcessor) parse(srcData []byte) (*WXBizMsg4Recv, *CryptError) {
	var msg4Recv WXBizMsg4Recv
	if err := xml.Unmarshal(srcData, &msg4Recv); err != nil {
		return nil, NewCryptError(ParseXmlError, "xml to msg fail")
	}
	return &msg4Recv, nil
}

func (XmlProcessor) serialize(msg4Send *WXBizMsg4Send) ([]byte, *CryptError) {
	xmlMsg, err := xml.Marshal(msg4Send)
	if err != nil {
		return nil, NewCryptError(GenXmlError, err.Error())
	}
	return xmlMsg, nil
}

func NewWXBizMsgCrypt(token, encodingAESKey, receiverID string, protocolType ProtocolType) *WXBizMsgCrypt {
	if protocolType != XmlType {
		panic("unsupported protocol")
	}
	return &WXBizMsgCrypt{
		token:             token,
		encodingAESKey:    encodingAESKey + "=",
		receiverID:        receiverID,
		protocolProcessor: new(XmlProcessor),
	}
}

func (c *WXBizMsgCrypt) randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func (c *WXBizMsgCrypt) pkcs7Padding(plaintext string, blockSize int) []byte {
	padding := blockSize - (len(plaintext) % blockSize)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	var buffer bytes.Buffer
	buffer.WriteString(plaintext)
	buffer.Write(padtext)
	return buffer.Bytes()
}

func (c *WXBizMsgCrypt) pkcs7Unpadding(plaintext []byte, blockSize int) ([]byte, *CryptError) {
	plaintextLen := len(plaintext)
	if plaintextLen == 0 {
		return nil, NewCryptError(DecryptAESError, "pkcs7 unpadding empty")
	}
	if plaintextLen%blockSize != 0 {
		return nil, NewCryptError(DecryptAESError, "pkcs7 block size")
	}
	paddingLen := int(plaintext[plaintextLen-1])
	return plaintext[:plaintextLen-paddingLen], nil
}

func (c *WXBizMsgCrypt) cbcEncrypter(plaintext string) ([]byte, *CryptError) {
	aeskey, err := base64.StdEncoding.DecodeString(c.encodingAESKey)
	if err != nil {
		return nil, NewCryptError(DecodeBase64Error, err.Error())
	}
	const blockSize = 32
	padMsg := c.pkcs7Padding(plaintext, blockSize)
	block, err := aes.NewCipher(aeskey)
	if err != nil {
		return nil, NewCryptError(EncryptAESError, err.Error())
	}
	ciphertext := make([]byte, len(padMsg))
	iv := aeskey[:aes.BlockSize]
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padMsg)
	base64Msg := make([]byte, base64.StdEncoding.EncodedLen(len(ciphertext)))
	base64.StdEncoding.Encode(base64Msg, ciphertext)
	return base64Msg, nil
}

func (c *WXBizMsgCrypt) cbcDecrypter(base64EncryptMsg string) ([]byte, *CryptError) {
	aeskey, err := base64.StdEncoding.DecodeString(c.encodingAESKey)
	if err != nil {
		return nil, NewCryptError(DecodeBase64Error, err.Error())
	}
	encryptMsg, err := base64.StdEncoding.DecodeString(base64EncryptMsg)
	if err != nil {
		return nil, NewCryptError(DecodeBase64Error, err.Error())
	}
	block, err := aes.NewCipher(aeskey)
	if err != nil {
		return nil, NewCryptError(DecryptAESError, err.Error())
	}
	if len(encryptMsg) < aes.BlockSize {
		return nil, NewCryptError(DecryptAESError, "encrypt_msg size invalid")
	}
	iv := aeskey[:aes.BlockSize]
	if len(encryptMsg)%aes.BlockSize != 0 {
		return nil, NewCryptError(DecryptAESError, "encrypt_msg block size")
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(encryptMsg, encryptMsg)
	return encryptMsg, nil
}

func (c *WXBizMsgCrypt) calSignature(timestamp, nonce, data string) string {
	sortArr := []string{c.token, timestamp, nonce, data}
	sort.Strings(sortArr)
	var buffer bytes.Buffer
	for _, value := range sortArr {
		buffer.WriteString(value)
	}
	sha := sha1.New()
	sha.Write(buffer.Bytes())
	return fmt.Sprintf("%x", sha.Sum(nil))
}

func (c *WXBizMsgCrypt) ParsePlainText(plaintext []byte) ([]byte, uint32, []byte, []byte, *CryptError) {
	const blockSize = 32
	plaintext, err := c.pkcs7Unpadding(plaintext, blockSize)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	textLen := uint32(len(plaintext))
	if textLen < 20 {
		return nil, 0, nil, nil, NewCryptError(IllegalBuffer, "plain too small")
	}
	random := plaintext[:16]
	msgLen := binary.BigEndian.Uint32(plaintext[16:20])
	if textLen < 20+msgLen {
		return nil, 0, nil, nil, NewCryptError(IllegalBuffer, "plain too small 2")
	}
	msg := plaintext[20 : 20+msgLen]
	receiverID := plaintext[20+msgLen:]
	return random, msgLen, msg, receiverID, nil
}

func (c *WXBizMsgCrypt) VerifyURL(msgSignature, timestamp, nonce, echostr string) ([]byte, *CryptError) {
	signature := c.calSignature(timestamp, nonce, echostr)
	if signature != msgSignature {
		return nil, NewCryptError(ValidateSignatureError, "signature not equal")
	}
	plaintext, err := c.cbcDecrypter(echostr)
	if err != nil {
		return nil, err
	}
	_, _, msg, receiverID, err := c.ParsePlainText(plaintext)
	if err != nil {
		return nil, err
	}
	if len(c.receiverID) > 0 && string(receiverID) != c.receiverID {
		return nil, NewCryptError(ValidateCorpidError, "receiver_id mismatch")
	}
	return msg, nil
}

func (c *WXBizMsgCrypt) EncryptMsg(replyMsg, timestamp, nonce string) ([]byte, *CryptError) {
	if timestamp == "" {
		timestamp = fmt.Sprintf("%d", rand.Int63())
	}
	if nonce == "" {
		nonce = c.randString(8)
	}
	randStr := c.randString(16)
	var buffer bytes.Buffer
	buffer.WriteString(randStr)
	msgLenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLenBuf, uint32(len(replyMsg)))
	buffer.Write(msgLenBuf)
	buffer.WriteString(replyMsg)
	buffer.WriteString(c.receiverID)
	tmpCiphertext, err := c.cbcEncrypter(buffer.String())
	if err != nil {
		return nil, err
	}
	ciphertext := string(tmpCiphertext)
	signature := c.calSignature(timestamp, nonce, ciphertext)
	msg4Send := NewWXBizMsg4Send(ciphertext, signature, timestamp, nonce)
	return c.protocolProcessor.serialize(msg4Send)
}

func (c *WXBizMsgCrypt) DecryptMsg(msgSignature, timestamp, nonce string, postData []byte) ([]byte, *CryptError) {
	msg4Recv, cryptErr := c.protocolProcessor.parse(postData)
	if cryptErr != nil {
		return nil, cryptErr
	}
	signature := c.calSignature(timestamp, nonce, msg4Recv.Encrypt)
	if signature != msgSignature {
		return nil, NewCryptError(ValidateSignatureError, "signature not equal")
	}
	plaintext, cryptErr := c.cbcDecrypter(msg4Recv.Encrypt)
	if cryptErr != nil {
		return nil, cryptErr
	}
	_, _, msg, receiverID, cryptErr := c.ParsePlainText(plaintext)
	if cryptErr != nil {
		return nil, cryptErr
	}
	if len(c.receiverID) > 0 && string(receiverID) != c.receiverID {
		return nil, NewCryptError(ValidateCorpidError, "receiver_id mismatch")
	}
	return msg, nil
}
