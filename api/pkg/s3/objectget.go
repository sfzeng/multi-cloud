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

package s3

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/opensds/multi-cloud/api/pkg/common"
	. "github.com/opensds/multi-cloud/api/pkg/s3/datatype"
	s3error "github.com/opensds/multi-cloud/s3/error"
	pb "github.com/opensds/multi-cloud/s3/proto"
	log "github.com/sirupsen/logrus"
	/*c "github.com/opensds/multi-cloud/api/pkg/context"
	. "github.com/opensds/multi-cloud/s3/pkg/exception"
	s3 "github.com/opensds/multi-cloud/s3/proto"
	"golang.org/x/net/context"
	*/
)

// supportedGetReqParams - supported request parameters for GET presigned request.
var supportedGetReqParams = map[string]string{
	"response-expires":             "Expires",
	"response-content-type":        "Content-Type",
	"response-cache-control":       "Cache-Control",
	"response-content-disposition": "Content-Disposition",
	"response-content-language":    "Content-Language",
	"response-content-encoding":    "Content-Encoding",
}

// setGetRespHeaders - set any requested parameters as response headers.
func setGetRespHeaders(w http.ResponseWriter, reqParams url.Values) {
	for k, v := range reqParams {
		if header, ok := supportedGetReqParams[k]; ok {
			w.Header()[header] = v
		}
	}
}

// Simple way to convert a func to io.Writer type.
type funcToWriter func([]byte) (int, error)

func (f funcToWriter) Write(p []byte) (int, error) {
	return f(p)
}

// GetObjectHandler - GET Object
func (s *APIService) ObjectGet(request *restful.Request, response *restful.Response) {
	bucketName := request.PathParameter("bucketName")
	objectKey := request.PathParameter("objectKey")
	rangestr := request.HeaderParameter("Range")
	log.Infof("%v\n", rangestr)

	ctx := common.InitCtxWithAuthInfo(request)
	objMetaRes, err := s.s3Client.GetObjectMeta(ctx, &pb.Object{BucketName: bucketName, ObjectKey: objectKey})
	if err != nil {
		log.Errorln("get object meta failed. err:", err)
		WriteErrorResponse(response, request, err)
		return
	}
	object := objMetaRes.Object

	// Get request range.
	var hrange *HttpRange
	rangeHeader := request.HeaderParameter("Range")
	if rangeHeader != "" {
		if hrange, err = ParseRequestRange(rangeHeader, object.Size); err != nil {
			// Handle only ErrorInvalidRange
			// Ignore other parse error and treat it as regular Get request like Amazon S3.
			if err == ErrorInvalidRange {
				WriteErrorResponse(response, request, s3error.ErrInvalidRange)
				return
			}
			// log the error.
			log.Errorln("invalid request range, err:", err)
		}
	}

	// Validate pre-conditions if any.
	if err = checkPreconditions(request.Request.Header, object); err != nil {
		// set object-related metadata headers
		response.AddHeader("Last-Modified", time.Unix(object.LastModified, 0).UTC().Format(http.TimeFormat))

		if object.Etag != "" {
			response.ResponseWriter.Header()["ETag"] = []string{"\"" + object.Etag + "\""}
		}
		if err == s3error.ContentNotModified { // write only header if is a 304
			WriteErrorResponseHeaders(response, err)
		} else {
			WriteErrorResponse(response, request, err)
		}
		return
	}

	// Get the object.
	startOffset := int64(0)
	length := object.Size
	if hrange != nil {
		startOffset = hrange.OffsetBegin
		length = hrange.GetLength()
	}
	stream, err := s.s3Client.GetObject(ctx, &pb.GetObjectInput{Bucket:bucketName, Key:objectKey, Offset: startOffset, Length: length,})
	if err != nil {
		log.Errorln("get object failed, err:%v", err)
		WriteErrorResponse(response, request, err)
		return
	}
	defer stream.Close()

	// Indicates if any data was written to the http.ResponseWriter
	dataWritten := false
	// io.Writer type which keeps track if any data was written.
	writer := funcToWriter(func(p []byte) (int, error) {
		if !dataWritten {
			// Set headers on the first write.
			// Set standard object headers.
			SetObjectHeaders(response, object, hrange)

			// Set any additional requested response headers.
			setGetRespHeaders(response.ResponseWriter, request.Request.URL.Query())
			dataWritten = true
		}
		n, err := response.Write(p)
		if n > 0 {

		}
		return n, err
	})

	s3err := int32(0)
	eof := false
	left := length
	for !eof && left > 0 {
		rsp, err := stream.Recv()
		if err != nil && err != io.EOF {
			log.Errorln("recv err", err)
			break
		}
		if err == io.EOF {
			eof = true
		}

		if rsp.ErrorCode != int32(s3error.ErrNoErr) {
			s3err = rsp.ErrorCode
			log.Errorf("received s3 service error, error code:", rsp.ErrorCode)
			break
		}
		if len(rsp.Data) == 0 {
			continue
		}
		_, err = writer.Write(rsp.Data)
		if err != nil {
			log.Errorln("failed to write data to client. err:", err)
			break
		}
		left -= int64(len(rsp.Data))
	}

	if !dataWritten {
		if s3err == int32(s3error.ErrNoErr) {
			writer.Write(nil)
		} else {
			WriteErrorResponse(response, request, s3error.S3ErrorCode(s3err))
			return
		}
	}

	log.Info("PUT object successfully.")
}
