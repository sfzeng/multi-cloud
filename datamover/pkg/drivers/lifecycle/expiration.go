// Copyright 2019 The OpenSDS Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lifecycle

import (
	"context"
	"errors"
	"strconv"

	"github.com/micro/go-micro/metadata"
	"github.com/opensds/multi-cloud/datamover/pkg/utils"
	"github.com/opensds/multi-cloud/datamover/proto"
	osdss3 "github.com/opensds/multi-cloud/s3/proto"
	log "github.com/sirupsen/logrus"
	"time"
	"github.com/opensds/multi-cloud/common/constants"
)

func doExpirationAction(acReq *datamover.LifecycleActionRequest) error {
	objKey := acReq.ObjKey
	bucketName := acReq.BucketName
	versionId := acReq.VersionId
	log.Infof("expiration action: objKey=%s, bucketName=%s, versionId=%s.\n", objKey, bucketName, versionId)

	if bucketName == "" {
		log.Infof("expiration of object[%s] is failed: virtual bucket is null.\n", objKey)
		return errors.New(utils.DMERR_InternalError)
	}

	// call API of s3 service to delete object
	delMetaReq := osdss3.DeleteObjectInput{Bucket: bucketName, Key: objKey, VersioId: versionId}
	// as expiration does not need to move data, so it's timeout time is not need to be too large.
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	ctx = metadata.NewContext(ctx, map[string]string{constants.CTX_KEY_IS_ADMIN: strconv.FormatBool(true)})
	_, err := s3client.DeleteObject(ctx, &delMetaReq)
	if err != nil {
		// if it is deleted failed this time, it will be delete again in the next schedule round
		log.Errorf("delete object [bucket:%s,objKey:%s] failed, err:%v\n",
			bucketName, objKey, err)
	} else {
		log.Infof("delete object [bucket:%s,objKey:%s] successfully.\n",
			bucketName, objKey)
	}

	return err
}
