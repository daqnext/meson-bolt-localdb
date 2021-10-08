package test

import (
	"fmt"
	ldb "github.com/daqnext/localfiledb"
	"go.etcd.io/bbolt"
	"log"
	"math/rand"
	"os"
	"sync"
	"testing"
)

var store *ldb.Store

type Pointer struct {
	Name string
}

type FileInfoWithIndex struct {
	HashKey        string `boltholdKey:"HashKey"`
	BindName       string `boltholdIndex:"BindName"`
	LastAccessTime int64  `boltholdIndex:"LastAccessTime"`
	FileSize       int64
	Rate           float64 `boltholdIndex:"Rate"`
	P              *Pointer
}

func Test_singleInsert(t *testing.T) {
	os.Remove("test.db")
	var err error

	store, err = ldb.Open("test.db", 0666, nil)
	if err != nil {
		log.Println("bolthold can't open")
	}

	p := &Pointer{"pointName"}
	fileInfo := FileInfoWithIndex{BindName: "bindName-1", LastAccessTime: int64(rand.Intn(100)), FileSize: int64(rand.Intn(100)), P: p}
	err = store.Insert("1", fileInfo)
	if err != nil {
		log.Println(err)
	}

	fileInfo = FileInfoWithIndex{BindName: "bindName-2", LastAccessTime: int64(rand.Intn(100)), FileSize: int64(rand.Intn(100)), P: p}
	err = store.Insert("2", fileInfo)
	if err != nil {
		log.Println(err)
	}
}

func Test_uniqueIndexInsert(t *testing.T) {
	os.Remove("test.db")
	var err error
	store, err = ldb.Open("test.db", 0666, nil)
	if err != nil {
		log.Println("bolthold can't open")
	}
	defer store.Close()

	type SomeStruct struct {
		Name string `boltholdIndex:"Name"`
		No   uint64 `boltholdUnique:"No"`
	}

	s := []SomeStruct{
		{"aaa", 1},
		{"bbb", 2},
		{"ccc", 1},
	}
	for i, v := range s {
		err := store.Insert(i, v)
		if err != nil {
			log.Println("insert index ", i, "err", err)
		}
	}

	var ss []*SomeStruct
	err = store.FindOne(&ss, nil)
	if err != nil {
		log.Println("FindOne query err", err)
	}
	for _, v := range ss {
		log.Println(v)
	}
}

func Test_batchInsert(t *testing.T) {
	os.Remove("test.db")
	var err error
	store, err = ldb.Open("test.db", 0666, nil)
	if err != nil {
		log.Println("bolthold can't open")
	}

	err = store.Bolt().Update(func(tx *bbolt.Tx) error {
		for i := 0; i < 100; i++ {
			hashKey := fmt.Sprintf("%d", i)
			bindName := fmt.Sprintf("bindname-%01d", rand.Intn(10)+4)
			p := &Pointer{"pointName"}
			fileInfo := FileInfoWithIndex{
				BindName:       bindName,
				LastAccessTime: int64(rand.Intn(100) - 50),
				FileSize:       int64(rand.Intn(100)),
				Rate:           float64(rand.Intn(1000))*0.33 - 150,
				P:              p}

			err := store.TxInsert(tx, hashKey, fileInfo)
			if err != nil {
				log.Println(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

func Test_singleGetByKey(t *testing.T) {
	Test_singleInsert(t)

	var info FileInfoWithIndex
	err := store.Get("1", &info)
	if err != nil {
		log.Println(err)
	}
	log.Println(info)

	var info2 FileInfoWithIndex
	err = store.Get("3", &info2)
	if err != nil {
		log.Println(err)
	}
	log.Println(info2)
}

func Test_queryGet(t *testing.T) {
	Test_batchInsert(t)

	var q *ldb.Query
	var qc *ldb.RangeCondition
	var err error
	_ = qc

	log.Println("query by primary key")
	var infos []*FileInfoWithIndex
	//KeyQuery
	qc = ldb.Ge("20").And(ldb.Le("20"))
	q = ldb.KeyQuery().Range(qc)
	err = store.Find(&infos, q)
	if err != nil {
		log.Println(err)
	}
	for _, v := range infos {
		log.Println(v)
	}

	log.Println("query by some index")
	var infos2 []*FileInfoWithIndex
	//IndexQuery
	qc = ldb.VPair(int64(-40), true, int64(-30), true).Or(ldb.VPair(int64(-10), true, int64(10), true)).Or(ldb.VPair(int64(30), true, int64(40), true))
	q = ldb.IndexQuery("LastAccessTime").Range(qc).Limit(10).Offset(0)
	err = store.Find(&infos2, q)
	if err != nil {
		log.Println(err)
	}
	for _, v := range infos2 {
		log.Println(v)
	}

	log.Println("query by some index")
	var infos3 []FileInfoWithIndex
	q = ldb.IndexQuery("Rate").Range(ldb.VPair(float64(-20), true, float64(20), true))
	err = store.Find(&infos3, q)
	if err != nil {
		log.Println(err)
	}
	for _, v := range infos3 {
		log.Println(v)
	}

	//log.Println("query by some index without range")
	//var infos4 []FileInfoWithIndex
	//q = ldb.IndexQuery("Rate").Offset(10).Limit(10)
	//err = store.Find(&infos4, q)
	//if err != nil {
	//	log.Println(err)
	//}
	//for _, v := range infos4 {
	//	log.Println(v)
	//}
	//
	//log.Println("query by some index without range")
	//var infos5 []FileInfoWithIndex
	//q = ldb.IndexQuery("LastAccessTime").Equal(int64(20))
	//err = store.Find(&infos5, q)
	//if err != nil {
	//	log.Println(err)
	//}
	//for _, v := range infos5 {
	//	log.Println(v)
	//}

}

func Test_updateQuery(t *testing.T) {
	Test_batchInsert(t)

	log.Println("update query")
	q := ldb.IndexQuery("LastAccessTime").Range(ldb.VPair(int64(10), true, int64(20), true))
	err := store.UpdateMatching(&FileInfoWithIndex{}, q, func(record interface{}) error {
		v, ok := record.(*FileInfoWithIndex)
		if !ok {
			log.Println("interface{} trans error")
		}
		v.FileSize = 999
		return nil
	})
	if err != nil {
		log.Println(err)
	}

	log.Println("query by primary key")
	var infos []FileInfoWithIndex
	q = ldb.KeyQuery().Range(ldb.VPair("0", true, "100", true))
	err = store.Find(&infos, q)
	if err != nil {
		log.Println(err)
	}
	for _, v := range infos {
		log.Println(v)
	}
}

func Test_deleteByPrimaryKey(t *testing.T) {
	Test_batchInsert(t)

	err := store.Delete("2", &FileInfoWithIndex{})
	if err != nil {
		log.Println(err)
	}

	store.Delete("5", &FileInfoWithIndex{})
	if err != nil {
		log.Println(err)
	}

	log.Println("query by primary key")
	var infos []FileInfoWithIndex
	q := ldb.KeyQuery().Range(ldb.VPair("0", true, "100", true))
	err = store.Find(&infos, q)
	if err != nil {
		log.Println(err)
	}
	for _, v := range infos {
		log.Println(v)
	}
}

func Test_deleteQuery(t *testing.T) {
	Test_batchInsert(t)

	log.Println("delete query")
	q := ldb.IndexQuery("LastAccessTime").Range(ldb.VPair(int64(10), true, int64(20), true))
	err := store.DeleteMatching(&FileInfoWithIndex{}, q)
	if err != nil {
		log.Println(err)
	}

	log.Println("query by primary key")
	var infos []FileInfoWithIndex
	q = ldb.KeyQuery().Range(ldb.VPair("0", true, "100", true))
	err = store.Find(&infos, q)
	if err != nil {
		log.Println(err)
	}
	for _, v := range infos {
		log.Println(v)
	}
}

func Test_checkIndexBucket(t *testing.T) {
	//os.Remove("test.db")
	var err error
	store, err = ldb.Open("test.db", 0666, nil)
	if err != nil {
		log.Println("bolthold can't open")
	}

	//
	store.Bolt().View(func(tx *bbolt.Tx) error {
		bk := tx.Bucket([]byte("_index:FileInfoWithIndex:BindName"))
		c := bk.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var key string
			var value [][]byte
			ldb.DefaultDecode(k, &key)
			ldb.DefaultDecode(v, &value)
			log.Println("key:", key, "value:", value)
		}

		bk = tx.Bucket([]byte("_index:FileInfoWithIndex:LastAccessTime"))
		c = bk.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var key int64
			var value [][]byte
			ldb.DefaultDecode(k, &key)
			ldb.DefaultDecode(v, &value)
			log.Println("key:", key, "value:", value)
		}

		bk = tx.Bucket([]byte("_index:FileInfoWithIndex:Rate"))
		c = bk.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var key float64
			var value [][]byte
			ldb.DefaultDecode(k, &key)
			ldb.DefaultDecode(v, &value)
			log.Println("key:", key, "value:", value)
		}

		return nil
	})
}

func Test_useSimpleKeyValue(t *testing.T) {
	var err error
	store, err = ldb.Open("test.db", 0666, nil)
	if err != nil {
		log.Println("bolthold can't open")
	}

	setV := map[string]int{}
	setV["a"] = 1
	setV["b"] = 2
	setV["c"] = 3

	store.Bolt().Update(func(tx *bbolt.Tx) error {
		bkt, _ := tx.CreateBucketIfNotExists([]byte("kvbuckt"))

		for k, v := range setV {
			vb, err := ldb.DefaultEncode(v)
			if err != nil {
				log.Println(err)
			} else {
				bkt.Put([]byte(k), vb)
			}
		}
		return nil
	})

	store.Bolt().View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte("kvbuckt"))
		keys := []string{"a", "b", "c"}
		for _, v := range keys {
			v1 := bkt.Get([]byte(v))
			var vv1 int
			err := ldb.DefaultDecode(v1, &vv1)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("key", v, "value", vv1)
			}
		}
		return nil
	})
}

func Test_reindex(t *testing.T) {
	var err error
	store, err = ldb.Open("test.db", 0666, nil)
	if err != nil {
		log.Println("bolthold can't open")
	}

	err = store.ReIndex(&FileInfoWithIndex{}, nil)
	if err != nil {
		log.Println(err)
	}

	store.RemoveIndex(&FileInfoWithIndex{}, "FileSize")
}

func mixUsage(round int) {
	for j := 0; j < 100; j++ {
		log.Println("start round ", j+1)

		//insert
		err := store.Bolt().Update(func(tx *bbolt.Tx) error {
			for i := 0; i < 10000; i++ {
				hashKey := fmt.Sprintf("%d", round*1000000+j*10000+i)
				bindName := fmt.Sprintf("bindname-%01d", rand.Intn(10000))
				p := &Pointer{"pointName"}
				fileInfo := FileInfoWithIndex{
					BindName:       bindName,
					LastAccessTime: int64(rand.Intn(100000)),
					FileSize:       int64(rand.Intn(1000000)),
					Rate:           float64(rand.Intn(100000))*0.33 - 15000,
					P:              p,
				}

				err := store.TxUpsert(tx, hashKey, fileInfo)
				if err != nil {
					log.Println("TxInsert err", err)
				}
			}
			return nil
		})
		if err != nil {
			log.Println("Update err", err)
		}

		//query
		qc := ldb.Gt(int64(10000)).And(ldb.Lt(int64(20000)))
		q := ldb.IndexQuery("LastAccessTime").Range(qc).Limit(1000).Offset(1000)
		var info []*FileInfoWithIndex
		err = store.Find(&info, q)
		if err != nil {
			log.Println("query find err", err)
		}

		//update
		l := fmt.Sprintf("%d", rand.Intn(1000000)+1000000)
		r := fmt.Sprintf("%d", rand.Intn(2000000)+2000000)
		qc = ldb.Gt(l).And(ldb.Lt(r))
		q = ldb.KeyQuery().Range(qc).Limit(1000).Offset(1000).Desc()
		err = store.UpdateMatching(&FileInfoWithIndex{}, q, func(record interface{}) error {
			v, ok := record.(*FileInfoWithIndex)
			if !ok {
				log.Println("interface{} trans error")
			}
			v.FileSize = 999
			return nil
		})
		if err != nil {
			log.Println("query UpdateMatching err", err)
		}

		//delete
		q = ldb.IndexQuery("Rate").Range(ldb.Ge(float64(-1000)).And(ldb.Le(float64(1000)))).Limit(1000)
		err = store.DeleteMatching(&FileInfoWithIndex{}, q)
		if err != nil {
			log.Println("query DeleteMatching err", err)
		}
	}
}

func Test_M(t *testing.T) {
	logFile, _ := os.Create("./log")
	log.SetOutput(logFile)

	os.Remove("test.db")
	var err error
	store, err = ldb.Open("test.db", 0666, nil)
	if err != nil {
		log.Println("bolthold can't open")
	}

	wg := &sync.WaitGroup{}
	wg.Add(10)
	for i := 0; i < 10; i++ {
		j := i
		go func() {
			mixUsage(j)
			wg.Done()
		}()
	}

	wg.Wait()
	log.Println("finish")
}

func Test_subBucket(t *testing.T) {
	os.Remove("test.db")
	var err error
	store, err = ldb.Open("test.db", 0666, nil)
	if err != nil {
		log.Println("bolthold can't open")
	}

	store.Bolt().Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("some bindname"))
		return err
	})

	store.Bolt().Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists([]byte("some bindname"))
		if err != nil {
			return err
		}

		hashKey := fmt.Sprintf("%d", 1)
		bindName := fmt.Sprintf("bindname-%01d", rand.Intn(10000))
		p := &Pointer{"pointName"}
		fileInfo := FileInfoWithIndex{
			BindName:       bindName,
			LastAccessTime: int64(rand.Intn(100000)),
			FileSize:       int64(rand.Intn(1000000)),
			Rate:           float64(rand.Intn(100000))*0.33 - 15000,
			P:              p,
		}
		err = store.UpsertBucket(bkt, hashKey, fileInfo)
		if err != nil {
			return err
		}

		var result FileInfoWithIndex
		err = store.GetFromBucket(bkt, "1", &result)
		if err != nil {
			return err
		}

		log.Println(result)

		var result2 []*FileInfoWithIndex
		q := ldb.KeyQuery().Range(ldb.Gt("0").And(ldb.Lt("10")))
		err = store.FindInBucket(bkt, &result2, q)
		if err != nil {
			return err
		}
		for _, v := range result2 {
			log.Println(v)
		}

		return nil
	})
}
