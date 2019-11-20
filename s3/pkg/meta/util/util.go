package util

import (
	"context"
	"encoding/hex"
	"github.com/micro/go-micro/metadata"
	"github.com/opensds/multi-cloud/api/pkg/common"
	. "github.com/opensds/multi-cloud/s3/error"
	log "github.com/sirupsen/logrus"
	"github.com/xxtea/xxtea-go/xxtea"
)

var XXTEA_KEY = []byte("hehehehe")

func Decrypt(value string) (string, error) {
	bytes, err := hex.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(xxtea.Decrypt(bytes, XXTEA_KEY)), nil
}

func Encrypt(value string) string {
	return hex.EncodeToString(xxtea.Encrypt([]byte(value), XXTEA_KEY))
}

func GetCredentialFromCtx(ctx context.Context) (isAdmin bool, tenantId string, err error) {
	var ok bool
	var md map[string]string
	md, ok = metadata.FromContext(ctx)
	if !ok {
		log.Error("get metadata from ctx failed.")
		err = ErrInternalError
		return
	}

	isAdmin = false
	isAdminStr, _ := md[common.CTX_KEY_IS_ADMIN]
	if isAdminStr == common.CTX_VAL_TRUE {
		isAdmin = true
	}

	if tenantId, ok = md[common.CTX_KEY_TENANT_ID]; !ok {
		log.Error("get tenantid failed.")
		err = ErrInternalError
		return
	}

	return
}
