// Copyright (c) 2018 Huawei Technologies Co., Ltd. All Rights Reserved.
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
package utils

import (
	"context"
	pb "github.com/opensds/multi-cloud/datamover/proto"
	flowtype "github.com/opensds/multi-cloud/dataflow/pkg/model"
)

type LocationInfo struct {
	StorType   string //aws-s3,azure-blob,hw-obs,ceph-s3 etc.
	Region     string
	EndPoint   string
	BucketName string //remote bucket name
	VirBucket  string //local bucket name
	Access     string
	Security   string
	BakendName string
}

type MoveWorker interface {
	DownloadObj(objKey string, srcLoca *LocationInfo, buf []byte) (size int64, err error)
	UploadObj(objKey string, destLoca *LocationInfo, buf []byte) error
	DeleteObj(objKey string, loca *LocationInfo) error
	MultiPartDownloadInit(srcLoca *LocationInfo) error
	DownloadRange(objKey string, srcLoca *LocationInfo, buf []byte, start int64, end int64) (size int64, err error)
	MultiPartUploadInit(objKey string, destLoca *LocationInfo) error
	UploadPart(objKey string, destLoca *LocationInfo, upBytes int64, buf []byte, partNumber int64, offset int64) error
	AbortMultipartUpload(objKey string, destLoca *LocationInfo) error
	CompleteMultipartUpload(objKey string, destLoca *LocationInfo) error
}

type Migration interface {
	Init()
	HandleMsg(msg string)
}

type AssistHandler interface {
	Init(in *string) error //in is json format
	Handle(ctx context.Context, in *pb.RunJobRequest, j *flowtype.Job, remoteBucket string) error //info include authentification and other handler need information
	//PostHandle(srcLoca *LocationInfo, destLoca *LocationInfo, j *flowtype.Job) error
}