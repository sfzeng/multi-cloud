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

package service

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/journeymidnight/yig/helper"
	"github.com/micro/go-micro/metadata"
	"github.com/opensds/multi-cloud/api/pkg/common"
	"github.com/opensds/multi-cloud/api/pkg/utils/constants"
	"github.com/opensds/multi-cloud/backend/proto"
	backendpb "github.com/opensds/multi-cloud/backend/proto"
	"github.com/opensds/multi-cloud/s3/error"
	. "github.com/opensds/multi-cloud/s3/error"
	dscommon "github.com/opensds/multi-cloud/s3/pkg/datastore/common"
	"github.com/opensds/multi-cloud/s3/pkg/datastore/driver"
	meta "github.com/opensds/multi-cloud/s3/pkg/meta/types"
	"github.com/opensds/multi-cloud/s3/pkg/meta/util"
	"github.com/opensds/multi-cloud/s3/pkg/utils"
	pb "github.com/opensds/multi-cloud/s3/proto"
	log "github.com/sirupsen/logrus"
)

var ChunkSize int = 2048

func (s *s3Service) CreateObject(ctx context.Context, in *pb.Object, out *pb.BaseResponse) error {
	log.Infoln("CreateObject is called in s3 service.")

	return nil
}

func (s *s3Service) UpdateObject(ctx context.Context, in *pb.Object, out *pb.BaseResponse) error {
	log.Infoln("PutObject is called in s3 service.")

	return nil
}

type StreamReader struct {
	in   pb.S3_PutObjectStream
	curr int
	req  *pb.PutObjectRequest
}

func (dr *StreamReader) Read(p []byte) (n int, err error) {
	left := len(p)
	for left > 0 {
		if dr.curr == 0 || (dr.req != nil && dr.curr == len(dr.req.Data)) {
			dr.req, err = dr.in.Recv()
			if err != nil && err != io.EOF {
				log.Errorln("failed to recv data with err:", err)
				return
			}
			if len(dr.req.Data) == 0 {
				log.Errorln("no data left to read.")
				err = io.EOF
				return
			}
			dr.curr = 0
		}

		copyLen := 0
		if len(dr.req.Data)-dr.curr > left {
			copyLen = left
		} else {
			copyLen = len(dr.req.Data) - dr.curr
		}
		log.Infoln("copy len:", copyLen)
		copy(p[n:], dr.req.Data[dr.curr:(dr.curr+copyLen)])
		dr.curr += copyLen
		left -= copyLen
		n += copyLen
	}
	return
}

func getBackend(ctx context.Context, backedClient backend.BackendService, bucket *meta.Bucket) (*backend.BackendDetail, error) {
	log.Infof("bucketName is %v:\n", bucket.Name)
	backendRep, backendErr := backedClient.ListBackend(ctx, &backendpb.ListBackendRequest{
		Offset: 0,
		Limit:  1,
		Filter: map[string]string{"name": bucket.DefaultLocation}})
	log.Infof("backendErr is %v:", backendErr)
	if backendErr != nil {
		log.Errorf("get backend %s failed.", bucket.DefaultLocation)
		return nil, backendErr
	}
	log.Infof("backendRep is %v:", backendRep)
	backend := backendRep.Backends[0]
	return backend, nil
}

func (s *s3Service) PutObject(ctx context.Context, in pb.S3_PutObjectStream) error {
	log.Infoln("PutObject is called in s3 service.")

	result := &pb.PutObjectResponse{}
	defer in.SendMsg(result)

	var ok bool
	var tenantId string
	var md map[string]string
	md, ok = metadata.FromContext(ctx)
	if !ok {
		log.Error("get metadata from ctx failed.")
		return ErrInternalError
	}

	if tenantId, ok = md[common.CTX_KEY_TENANT_ID]; !ok {
		log.Error("get tenantid failed.")
		return ErrInternalError
	}

	obj := &pb.Object{}
	err := in.RecvMsg(obj)
	if err != nil {
		log.Errorln("failed to get msg with err:", err)
		return ErrInternalError
	}
	obj.TenantId = tenantId
	log.Infof("metadata of object is:%+v\n", obj)
	log.Infof("*********bucket:%s,key:%s,size:%d\n", obj.BucketName, obj.ObjectKey, obj.Size)

	bucket, err := s.MetaStorage.GetBucket(ctx, obj.BucketName, true)
	if err != nil {
		log.Errorln("get bucket failed with err:", err)
		return err
	}
	if bucket == nil {
		log.Infoln("bucket is nil")
	}

	switch bucket.Acl.CannedAcl {
	case "public-read-write":
		break
	default:
		if bucket.TenantId != obj.TenantId {
			return s3error.ErrBucketAccessForbidden
		}
	}

	data := &StreamReader{in: in}
	var limitedDataReader io.Reader
	if obj.Size > 0 { // request.ContentLength is -1 if length is unknown
		limitedDataReader = io.LimitReader(data, obj.Size)
	} else {
		limitedDataReader = data
	}

	backend, err := getBackend(ctx, s.backendClient, bucket)
	if err != nil {
		log.Errorln("failed to get backend client with err:", err)
		return err
	}

	log.Infoln("bucket location:", obj.Location, " backendtype:", backend.Type,
		" endpoint:", backend.Endpoint)
	bodyMd5 := md["md5Sum"]
	ctx = context.Background()
	ctx = context.WithValue(ctx, dscommon.CONTEXT_KEY_SIZE, obj.Size)
	ctx = context.WithValue(ctx, dscommon.CONTEXT_KEY_MD5, bodyMd5)
	sd, err := driver.CreateStorageDriver(backend.Type, backend)
	if err != nil {
		log.Errorln("failed to create storage. err:", err)
		return nil
	}
	res, err := sd.Put(ctx, limitedDataReader, obj)
	if err != nil {
		log.Errorln("failed to put data. err:", err)
		return err
	}
	oid := res.ObjectId
	bytesWritten := res.Written

	// Should metadata update failed, add `maybeObjectToRecycle` to `RecycleQueue`,
	// so the object in Ceph could be removed asynchronously
	//maybeObjectToRecycle := objectToRecycle{
	//	location: cephCluster.Name,
	//	pool:     poolName,
	//	objectId: oid,
	//}
	if bytesWritten < obj.Size {
		//RecycleQueue <- maybeObjectToRecycle
		log.Warnf("write objects, already written(%d), total size(%d)\n", bytesWritten, obj.Size)
	}
	result.Md5 = res.Etag

	/*if signVerifyReader, ok := data.(*signature.SignVerifyReader); ok {
		credential, err = signVerifyReader.Verify()
		if err != nil {
			RecycleQueue <- maybeObjectToRecycle
			return
		}
	}*/
	// TODO validate bucket policy and fancy ACL
	obj.ObjectId = oid
	obj.LastModified = time.Now().UTC().Unix()
	obj.Etag = res.Etag
	obj.ContentType = md["Content-Type"]
	obj.DeleteMarker = false
	obj.CustomAttributes = md /* TODO: only reserve http header attr*/
	obj.Type = meta.ObjectTypeNormal
	obj.Tier = utils.Tier1 // Currently only support tier1
	obj.StorageMeta = res.Meta

	object := &meta.Object{Object: obj}

	result.LastModified = object.LastModified
	//var nullVerNum uint64
	//nullVerNum, err = yig.checkOldObject(bucketName, objectName, bucket.Versioning)
	//if err != nil {
	//	RecycleQueue <- maybeObjectToRecycle
	//	return
	//}
	//if bucket.Versioning == "Enabled" {
	//	result.VersionId = object.GetVersionId()
	//}
	//// update null version number
	//if bucket.Versioning == "Suspended" {
	//	nullVerNum = uint64(object.LastModifiedTime.UnixNano())
	//}
	//
	//if nullVerNum != 0 {
	//	objMap := &meta.ObjMap{
	//		Name:       objectName,
	//		BucketName: bucketName,
	//	}
	//	err = yig.MetaStorage.PutObject(object, nil, objMap, true)
	//} else {
	//	err = yig.MetaStorage.PutObject(object, nil, nil, true)
	//}

	err = s.MetaStorage.PutObject(ctx, object, nil, nil, true)
	if err != nil {
		log.Errorln("failed to put object meta. err:", err)
		//RecycleQueue <- maybeObjectToRecycle
		return ErrDBError
	}
	if err == nil {
		//b.MetaStorage.Cache.Remove(redis.ObjectTable, obj.OBJECT_CACHE_PREFIX, bucketName+":"+objectKey+":")
		//b.DataCache.Remove(bucketName + ":" + objectKey + ":" + object.GetVersionId())
	}

	return nil
}

func (s *s3Service) GetObjectMeta(ctx context.Context, in *pb.Object, out *pb.GetObjectMetaResult) error {
	var err error
	defer func() {
		out.ErrorCode = GetErrCode(err)
	} ()

	object, err := s.MetaStorage.GetObject(ctx, in.BucketName, in.ObjectKey, true)
	if err != nil {
		log.Errorln("failed to get object info from meta storage. err:", err)
		return err
	}
	out.Object = object.Object
	return nil
}

func (s *s3Service) GetObject(ctx context.Context, req *pb.GetObjectInput, stream pb.S3_GetObjectStream) error {
	log.Infoln("GetObject is called in s3 service.")
	bucketName := req.Bucket
	objectName := req.Key
	offset := req.Offset
	length := req.Length

	var err error
	getObjRes := &pb.GetObjectResponse{}
	defer func ()  {
		getObjRes.ErrorCode = GetErrCode(err)
		stream.SendMsg(getObjRes)
	} ()

	object, err := s.MetaStorage.GetObject(ctx, bucketName, objectName, true)
	if err != nil {
		log.Errorln("failed to get object info from meta storage. err:", err)
		return err
	}

	bucket, err := s.MetaStorage.GetBucket(ctx, bucketName, true)
	if err != nil {
		log.Errorln("failed to get bucket from meta storage. err:", err)
		return err
	}

	// if this object has only one part
	backend, err := getBackend(ctx, s.backendClient, bucket)
	if err != nil {
		log.Errorln("unable to get backend. err:", err)
		return err
	}
	sd, err := driver.CreateStorageDriver(backend.Type, backend)
	if err != nil {
		log.Errorln("failed to create storage driver. err:", err)
		return nil
	}
	log.Infof("get object offset %v, length %v", offset, length)
	reader, err := sd.Get(ctx, object.Object, offset, offset + length - 1)
	if err != nil {
		log.Errorln("failed to get data. err:", err)
		return err
	}

	eof := false
	left := object.Size
	buf := make([]byte, ChunkSize)
	for !eof && left > 0 {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			log.Errorln("failed to read, err:", err)
			break
		}
		if err == io.EOF {
			log.Debugln("finished read")
			eof = true
		}
		if n == 0 {
			break
		}

		err = stream.Send(&pb.GetObjectResponse{ErrorCode:int32(ErrNoErr), Data:buf[0:n]})
		if err != nil {
			log.Infof("stream send error: %v\n", err)
			break
		}
		left -= int64(n)
	}

	log.Infoln("get object successfully")
	return nil
}

func (s *s3Service) DeleteObject(ctx context.Context, in *pb.DeleteObjectInput, out *pb.BaseResponse) error {
	log.Infoln("DeleteObject is called in s3 service.")

	return nil
}

func (s *s3Service) ListObjects(ctx context.Context, in *pb.ListObjectsRequest, out *pb.ListObjectsResponse) error {
	log.Infof("ListObject is called in s3 service, bucket is %s.\n", in.Bucket)
	var err error
	defer func() {
		out.ErrorCode = GetErrCode(err)
	}()
	// Check ACL
	bucket, err := s.MetaStorage.GetBucket(ctx, in.Bucket, true)
	if err != nil {
		log.Errorf("err:%v\n", err)
		return nil
	}

	isAdmin, tenantId, err := util.GetCredentialFromCtx(ctx)
	if err != nil && isAdmin == false {
		log.Error("get tenant id failed")
		err = ErrInternalError
		return nil
	}

	// administrator can get any resource
	if isAdmin == false {
		switch bucket.Acl.CannedAcl {
		case "public-read", "public-read-write":
			break
		case "authenticated-read":
			if tenantId == "" {
				log.Error("no tenant id provided")
				err = ErrBucketAccessForbidden
				return nil
			}
		default:
			if bucket.TenantId != tenantId {
				log.Errorf("tenantId(%s) does not much bucket.TenantId(%s)", tenantId, bucket.TenantId)
				err = ErrBucketAccessForbidden
				return nil
			}
		}
		// TODO validate user policy and ACL
	}

	retObjects, prefixes, truncated, nextMarker, _, err := s.ListObjectsInternal(ctx, in)
	if truncated && len(nextMarker) != 0 {
		out.NextMarker = nextMarker
	}
	if in.Version == constants.ListObjectsType2Int {
		out.NextMarker = util.Encrypt(out.NextMarker)
	}

	objects := make([]*pb.Object, 0, len(retObjects))
	for _, obj := range retObjects {
		object := pb.Object{
			LastModified: obj.LastModified,
			Etag:         obj.Etag,
			Size:         obj.Size,
			Tier:         obj.Tier,
			Location:     obj.Location,
			TenantId:     obj.TenantId,
			BucketName:   obj.BucketName,
		}
		if in.EncodingType != "" { // only support "url" encoding for now
			object.ObjectKey = url.QueryEscape(obj.ObjectKey)
		} else {
			object.ObjectKey = obj.ObjectKey
		}
		object.StorageClass, _ = GetNameFromTier(obj.Tier, utils.OSTYPE_OPENSDS)
		objects = append(objects, &object)
		log.Infof("object:%+v\n", object)
	}
	out.Objects = objects
	out.Prefixes = prefixes
	out.IsTruncated = truncated

	if in.EncodingType != "" { // only support "url" encoding for now
		out.Prefixes = helper.Map(out.Prefixes, func(s string) string {
			return url.QueryEscape(s)
		})
		out.NextMarker = url.QueryEscape(out.NextMarker)
	}

	err = ErrNoErr
	return nil
}

func (s *s3Service) ListObjectsInternal(ctx context.Context, request *pb.ListObjectsRequest) (retObjects []*meta.Object,
	prefixes []string, truncated bool, nextMarker, nextVerIdMarker string, err error) {
	log.Infoln("Prefix:", request.Prefix, "Marker:", request.Marker, "MaxKeys:",
		request.MaxKeys, "Delimiter:", request.Delimiter, "Version:", request.Version,
		"keyMarker:", request.KeyMarker, "versionIdMarker:", request.VersionIdMarker)

	filt := make(map[string]string)
	if request.Versioned {
		filt[common.KMarker] = request.KeyMarker
		filt[common.KVerMarker] = request.VersionIdMarker
	} else if request.Version == constants.ListObjectsType2Int {
		if request.ContinuationToken != "" {
			var marker string
			marker, err = util.Decrypt(request.ContinuationToken)
			if err != nil {
				err = ErrInvalidContinuationToken
				return
			}
			filt[common.KMarker] = marker
		} else {
			filt[common.KMarker] = request.StartAfter
		}
	} else { // version 1
		filt[common.KMarker] = request.Marker
	}

	filt[common.KPrefix] = request.Prefix
	filt[common.KDelimiter] = request.Delimiter
	// currentlly, request.Filter only support filter by 'lastmodified' and 'tier'
	for k, v := range request.Filter {
		filt[k] = v
	}

	return s.MetaStorage.Db.ListObjects(ctx, request.Bucket, request.Versioned, int(request.MaxKeys), filt)
}
