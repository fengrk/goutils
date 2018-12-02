package httputils

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/url"
)

func GetDomain(uri string) (content string, err error) {
	uriInfo, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	// todo not finish
	return uriInfo.Scheme + "://" + uriInfo.Host + uriInfo.Path, nil
}

func GetMd5(content string) string {
	md5Writer := md5.New()
	io.WriteString(md5Writer, content)
	return fmt.Sprintf("%x", md5Writer.Sum(nil))
}
