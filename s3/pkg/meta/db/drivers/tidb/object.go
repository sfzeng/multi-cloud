package tidbclient

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"math"
	"strconv"
	"time"

	. "github.com/opensds/multi-cloud/s3/pkg/meta/types"
	pb "github.com/opensds/multi-cloud/s3/proto"
	log "github.com/sirupsen/logrus"
	"github.com/xxtea/xxtea-go/xxtea"
)

func (t *TidbClient) GetObject(ctx context.Context, bucketName, objectName, version string) (object *Object, err error) {
	var sqltext, ibucketname, iname, customattributes, acl, lastModified string
	var iversion uint64
	var row *sql.Row
	if version == "" {
		sqltext = "select bucketname,name,version,location,tenantid,userid,size,objectid,lastmodifiedtime,etag," +
			"contenttype,customattributes,acl,nullversion,deletemarker,ssetype,encryptionkey,initializationvector,type,tier,storageMeta" +
			" from objects where bucketname=? and name=? order by bucketname,name,version limit 1;"
		row = t.Client.QueryRow(sqltext, bucketName, objectName)
	} else {
		sqltext = "select bucketname,name,version,location,tenantid,userid,size,objectid,lastmodifiedtime,etag," +
			"contenttype,customattributes,acl,nullversion,deletemarker,ssetype,encryptionkey,initializationvector,type,tier,storageMeta" +
			" from objects where bucketname=? and name=? and version=?;"
		row = t.Client.QueryRow(sqltext, bucketName, objectName, version)
	}
	log.Infof("sqltext:%s, version:%s\n", sqltext, version)
	object = &Object{Object: &pb.Object{ServerSideEncryption: &pb.ServerSideEncryption{}}}
	err = row.Scan(
		&ibucketname,
		&iname,
		&iversion,
		&object.Location,
		&object.TenantId,
		&object.UserId,
		&object.Size,
		&object.ObjectId,
		&lastModified,
		&object.Etag,
		&object.ContentType,
		&customattributes,
		&acl,
		&object.NullVersion,
		&object.DeleteMarker,
		&object.ServerSideEncryption.SseType,
		&object.ServerSideEncryption.EncryptionKey,
		&object.ServerSideEncryption.InitilizationVector,
		&object.Type,
		&object.Tier,
		&object.StorageMeta,
	)
	if err != nil {
		log.Errorf("err: %v\n", err)
		err = handleDBError(err)
		return
	}

	object.ObjectKey = objectName
	object.BucketName = bucketName
	lastModifiedTime, _ := time.ParseInLocation(TIME_LAYOUT_TIDB, lastModified, time.Local)
	object.LastModified = lastModifiedTime.Unix()

	err = json.Unmarshal([]byte(acl), &object.Acl)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(customattributes), &object.CustomAttributes)
	if err != nil {
		return
	}
	// TODO: getting multi-parts
	timestamp := math.MaxUint64 - iversion
	timeData := []byte(strconv.FormatUint(timestamp, 10))
	object.VersionId = hex.EncodeToString(xxtea.Encrypt(timeData, XXTEA_KEY))
	return
}

func (t *TidbClient) PutObject(ctx context.Context, object *Object, tx interface{}) (err error) {
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
	sql, args := object.GetCreateSql()
	_, err = sqlTx.Exec(sql, args...)
	// TODO: multi-part handle, see issue https://github.com/opensds/multi-cloud/issues/690

	return err
}

func (t *TidbClient) DeleteObject(ctx context.Context, object *Object, tx interface{}) (err error) {
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

	v := math.MaxUint64 - uint64(object.LastModified)
	version := strconv.FormatUint(v, 10)
	sqltext := "delete from objects where name=? and bucketname=? and version=?;"
	_, err = sqlTx.Exec(sqltext, object.ObjectKey, object.BucketName, version)
	if err != nil {
		return err
	}
	sqltext = "delete from objectpart where objectname=? and bucketname=? and version=?;"
	_, err = sqlTx.Exec(sqltext, object.ObjectKey, object.BucketName, version)
	if err != nil {
		return err
	}
	return nil
}
