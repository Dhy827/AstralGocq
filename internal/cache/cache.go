// Package cache impl the cache for gocq
package cache

import (
	pb "github.com/ProtocolScience/AstralGo/client/pb/database"
	"github.com/RomiChan/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"sync"
	"time"
)

// Media Cache DBs
var (
	Media Cache
)

// Cache wraps the btree.DB for concurrent safe
type Cache struct {
	ldb           *leveldb.DB
	clean         sync.Mutex
	lastCleanTime uint32
}

// ScanExpiredData 扫描并删除过期数据，防止gocq的数据库越来越肥硕
func (c *Cache) ScanExpiredData() {
	// 创建迭代器
	c.clean.Lock()
	curTime := uint32(time.Now().Unix())
	if curTime-c.lastCleanTime > 3600 {
		c.lastCleanTime = curTime
		iter := c.ldb.NewIterator(nil, nil)
		for iter.Next() {
			value := iter.Value()
			result := pb.DatabaseRecord{}
			if proto.Unmarshal(value, &result) == nil {
				if curTime > result.Register.ExpiredTime {
					_ = c.ldb.Delete(iter.Key(), nil)
				}
			}
		}
		iter.Release()
	}
	defer c.clean.Unlock()
}

// Insert 添加媒体缓存
func (c *Cache) Insert(md5 []byte, record *pb.DatabaseRecord) {
	c.ScanExpiredData()
	record.Register = &pb.DataRegister{
		ExpiredTime: uint32(time.Now().Unix()) + 2592000,
	}
	data, _ := proto.Marshal(record)
	_ = c.ldb.Put(md5, data, nil)
}

// Get 获取缓存信息
func (c *Cache) Get(md5 []byte) *pb.DatabaseRecord {
	c.ScanExpiredData()
	got, _ := c.ldb.Get(md5, nil)
	result := pb.DatabaseRecord{}
	err := proto.Unmarshal(got, &result)
	if err != nil {
		return nil
	}
	c.Insert(md5, &result) //update time
	return &result
}

// Delete 删除指定缓存
func (c *Cache) Delete(md5 []byte) {
	_ = c.ldb.Delete(md5, nil)
}

// Init 初始化 Cache
func Init() {
	open := func(typ, path string, cache *Cache) {
		ldb, err := leveldb.OpenFile(path, &opt.Options{
			WriteBuffer: 4 * opt.KiB,
		})
		if err != nil {
			log.Fatalf("open cache %s db failed: %v", typ, err)
		}
		cache.ldb = ldb
	}
	open("database", "data/database", &Media)
}
