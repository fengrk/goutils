package common

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/frkhit/gozbarlib"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"strings"
)

var tempDir = os.TempDir()

func Base64ToImageFile(base64Content string, targetFile string) (err error) {
	rawImage := base64Content[strings.Index(base64Content, ",")+1:]
	nameList := strings.Split(targetFile, ".")
	fileType := nameList[len(nameList)-1]
	if fileType != "png" && fileType != "jpeg" {
		return fmt.Errorf("No support for " + fileType + " image")
	}
	
	// Encoded Image DataUrl //
	data, err := base64.StdEncoding.DecodeString(rawImage)
	if err != nil {
		return err
	}
	res := bytes.NewReader(data)
	imageFile, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer imageFile.Close()
	
	if fileType == "png" {
		pngI, pngErr := png.Decode(res)
		if pngErr != nil {
			return pngErr
		}
		png.Encode(imageFile, pngI)
	} else
	{
		jpgI, jpgErr := jpeg.Decode(res)
		if jpgErr != nil {
			return jpgErr
		}
		jpeg.Encode(imageFile, jpgI, &jpeg.Options{Quality: 75})
	}
	
	return nil
}

func ReadQRFromBase64(base64 string) (content string, err error) {
	fileName := path.Join(tempDir, GetUUID()+"."+base64[11:strings.Index(base64, ";")])
	
	err = Base64ToImageFile(base64, fileName)
	if err != nil {
		return "", err
	}
	
	defer os.Remove(fileName)
	
	return gozbarlib.QRCodeReader(fileName)
	
}

func GetPath(name string) string {
	if FileExists(name) {
		return name
	}
	if strings.Index(name, ":") > -1 && len(strings.Split(name, ":")) == 2 {
		pathList := strings.Split(name, ":")
		newName := path.Join("/mnt/", strings.ToLower(pathList[0])+pathList[1])
		if FileExists(newName) {
			return newName
		}
	}
	return name
}
