package utils

import (
	"bytes"
	"encoding/base64"
	"io"

	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

// Get QR code from JSON string.
//
//	@param json
//	@return *string
//	@return error
func GetQrBytesFromJson(json string) (*string, error) {
	// Create code
	qrc, err := qrcode.NewWith(json,
		qrcode.WithEncodingMode(qrcode.EncModeByte),
		qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionQuart),
	)
	if err != nil {
		return nil, err
	}
	// Get bytes
	buf := bytes.NewBuffer(nil)
	wr := nopCloser{Writer: buf}
	w2 := standard.NewWithWriter(wr, standard.WithQRWidth(40))
	if err = qrc.Save(w2); err != nil {
		panic(err)
	}
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	return &base64Str, nil
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
