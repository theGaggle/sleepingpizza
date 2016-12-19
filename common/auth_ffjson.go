// DO NOT EDIT!
// Code generated by ffjson <https://github.com/pquerna/ffjson>
// source: auth.go
// DO NOT EDIT!

package common

import (
	fflib "github.com/pquerna/ffjson/fflib/v1"
)

func (mj *Captcha) MarshalJSON() ([]byte, error) {
	var buf fflib.Buffer
	if mj == nil {
		buf.WriteString("null")
		return buf.Bytes(), nil
	}
	err := mj.MarshalJSONBuf(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func (mj *Captcha) MarshalJSONBuf(buf fflib.EncodingBuffer) error {
	if mj == nil {
		buf.WriteString("null")
		return nil
	}
	var err error
	var obj []byte
	_ = obj
	_ = err
	buf.WriteString(`{"Captcha":`)
	fflib.WriteJsonString(buf, string(mj.Captcha))
	buf.WriteString(`,"CaptchaID":`)
	fflib.WriteJsonString(buf, string(mj.CaptchaID))
	buf.WriteByte('}')
	return nil
}