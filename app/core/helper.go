package core

import (
	"os"
	"strings"
)

func GetUploadFilepath() string {
	uploadFilepath := Config.Server.UploadFilepath
	if uploadFilepath == "" {
		uploadFilepath = "uploads"
	}
	if !strings.HasSuffix(uploadFilepath, "/") {
		uploadFilepath += "/"
	}
	os.MkdirAll(uploadFilepath, 0777)
	return uploadFilepath
}
