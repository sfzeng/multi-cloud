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
	"github.com/opensds/multi-cloud/api/pkg/common"
	. "github.com/opensds/multi-cloud/datamover/pkg/utils"
	datamover "github.com/opensds/multi-cloud/datamover/proto"
	osdss3 "github.com/opensds/multi-cloud/s3/proto"
	log "github.com/sirupsen/logrus"
)

func deleteObj(objKey, virtBucket, versionId string) error {
	log.Infof("object expiration: objKey=%s, virtBucket=%s, versionId=%s\n", objKey, virtBucket, versionId)
	if virtBucket == "" {
		log.Infof("expiration of object[%s] is failed: virtual bucket is null.\n", objKey)
		return errors.New(DMERR_InternalError)
	}

	// call API of s3 service to delete object
	delMetaReq := osdss3.DeleteObjectInput{Bucket: virtBucket, Key: objKey, VersioId: versionId}
	ctx := metadata.NewContext(context.Background(), map[string]string{
		common.CTX_KEY_IS_ADMIN:  strconv.FormatBool(true),
		common.CTX_KEY_TENANT_ID: INTERNAL_TENANT,
	})
	_, err := s3client.DeleteObject(ctx, &delMetaReq)
	if err != nil {
		// if it is deleted failed this time, it will be delete again in the next schedule round
		log.Errorf("delete object metadata of obj[bucket:%s,objKey:%s] failed, err:%v\n",
			virtBucket, objKey, err)
		return err
	} else {
		log.Infof("delete object metadata of obj[bucket:%s,objKey:%s] successfully.\n",
			virtBucket, objKey)
	}

	return err
}

func doExpirationAction(acReq *datamover.LifecycleActionRequest) error {
	log.Infof("expiration action: delete %s.\n", acReq.ObjKey)

	err := deleteObj(acReq.ObjKey, acReq.BucketName, acReq.VersionId)
	if err != nil {
		log.Errorf("expiration execute failed, err:%v\n", err)
	} else {
		log.Infof("expiration execute suceed, obj:%s\n", acReq.ObjKey)
	}

	return err
}
