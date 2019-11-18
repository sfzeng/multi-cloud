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
package tidbclient

import (
	"context"
	"database/sql"
	. "github.com/opensds/multi-cloud/s3/pkg/meta/types"
	log "github.com/sirupsen/logrus"
	"math"
	"time"
)

//gc
func (t *TidbClient) PutObjectToGarbageCollection(ctx context.Context, object *Object, tx interface{}) (err error) {
	var sqlTx *sql.Tx
	if tx == nil {
		tx, err = t.Client.Begin()
		defer func() {
			if err == nil {
				err = sqlTx.Commit()
			}
			if err != nil {
				sqlTx.Rollback()
			}
		}()
	}
	sqlTx, _ = tx.(*sql.Tx)

	o := GarbageCollectionFromObject(object)
	var hasPart bool
	if len(o.Parts) > 0 {
		hasPart = true
	}
	mtime := o.MTime.Format(TIME_LAYOUT_TIDB)
	version := math.MaxUint64 - uint64(object.LastModified)
	sqltext := "insert ignore into gc(bucketname,objectname,version,location,storagemeta,objectid,status,mtime,part,triedtimes) values(?,?,?,?,?,?,?,?,?,?);"
	_, err = sqlTx.Exec(sqltext, o.BucketName, o.ObjectName, version, o.Location, o.StorageMeta, o.ObjectId, o.Status, mtime, hasPart, o.TriedTimes)
	if err != nil {
		return err
	}
	// TODO: multipart upload
	/*for _, p := range object.Parts {
		psql, args := p.GetCreateGcSql(o.BucketName, o.ObjectName, version)
		_, err = sqlTx.Exec(psql, args...)
		if err != nil {
			return err
		}
	}*/

	return nil
}

func GarbageCollectionFromObject(o *Object) (gc GarbageCollection) {
	gc.BucketName = o.BucketName
	gc.ObjectName = o.ObjectKey
	gc.Location = o.Location
	gc.StorageMeta = o.StorageMeta
	gc.ObjectId = o.ObjectId
	gc.Status = "Pending"
	gc.MTime = time.Now().UTC()
	// TODO: multipart upload
	//gc.Parts = o.Parts
	gc.TriedTimes = 0
	return
}

func (t *TidbClient) PutGcobjRecord(ctx context.Context, o *Object, tx interface{}) (err error) {
	var sqlTx *sql.Tx
	if tx == nil {
		tx, err = t.Client.Begin()
		defer func() {
			if err == nil {
				err = sqlTx.Commit()
			}
			if err != nil {
				sqlTx.Rollback()
			}
		}()
	}
	sqlTx, _ = tx.(*sql.Tx)

	version := math.MaxUint64 - uint64(o.LastModified)
	lastModifiedTime := time.Unix(o.LastModified, 0).Format(TIME_LAYOUT_TIDB)
	sqltext := "insert into gcobjs (bucketname, name, version, location, tenantid, userid, size, objectid, " +
		" lastmodifiedtime, etag, tier, storageMeta) values(?,?,?,?,?,?,?,?,?,?,?,?)"
	args := []interface{}{o.BucketName, o.ObjectKey, version, o.Location, o.TenantId, o.UserId, o.Size, o.ObjectId,
		lastModifiedTime, o.Etag, o.Tier, o.StorageMeta}
	log.Debugf("sqltext:%s, args:%v\n", sqltext, args)
	_, err = sqlTx.Exec(sqltext, args...)
	log.Debugf("err:%v\n", err)

	return err
}

func (t *TidbClient) DeleteGcobjRecord(ctx context.Context, o *Object, tx interface{}) (err error) {
	var sqlTx *sql.Tx
	if tx == nil {
		tx, err = t.Client.Begin()
		defer func() {
			if err == nil {
				err = sqlTx.Commit()
			}
			if err != nil {
				sqlTx.Rollback()
			}
		}()
	}
	sqlTx, _ = tx.(*sql.Tx)

	version := math.MaxUint64 - uint64(o.LastModified)

	sqltext := "delete from gcobjs where bucketname=? and name=? and version=?"
	args := []interface{}{o.BucketName, o.ObjectKey, version}
	log.Debugf("sqltext:%s, args:%v\n", sqltext, args)
	_, err = sqlTx.Exec(sqltext, args...)
	log.Debugf("err:%v\n", err)

	return err
}
