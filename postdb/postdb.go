package postdb

import (
	"context"
	"github.com/redis/go-redis/v9"
	"os"
	"github.com/stalltrix/kepweb/postcodec"
	"github.com/stalltrix/kep-demo/logger"
	"sync"
	"sync/atomic"
	"time"
	"math/rand"
	"path/filepath"
	"encoding/gob"
	"io"
	"strconv"
)

type DataHandle struct {
	ctx context.Context
	rdb *redis.Client
}

type loadData struct {
	data *postcodec.Post
	last int64
}

var (
	log logger.Log_TYPE
	foundMap sync.Map
	size atomic.Int64
	lock atomic.Bool
	rnd *rand.Rand
	last_clean int64
	is_init bool
	local_path string
	is_redis bool
	localLock []sync.RWMutex
)

func Open(path,ipaddr,passwd string) (*DataHandle, error) {
	if is_init {
		return nil,os.ErrExist
	}
	if ipaddr==""{
		if path =="" {
			return nil,os.ErrNotExist
		}
		ok:=fileTest(path)
		if !ok {
			return nil,os.ErrPermission
		}
		local_path=path
		is_init=true
		localLock=make([]sync.RWMutex,256)
		log.Println("Warn: local cache set, local cache only use for development environment")
		return &DataHandle{},nil
	}
	ctx := context.Background()
    rdb := redis.NewClient(&redis.Options{
        Addr:     ipaddr,
        Password: passwd,
        DB:       0,
    })
	_, err := rdb.Ping(ctx).Result()
    if err != nil {
       return nil,err
    }
	st:=&DataHandle{
		ctx:ctx,
		rdb:rdb,
	}
	is_redis=true
	is_init=true
	return st,nil
}

func (s *DataHandle) Store(hex string,p *postcodec.Post) {
	found:=&loadData{
		data: p,
		last: time.Now().Unix(),
	}
	_,loaded := foundMap.LoadOrStore(hex,found)
    if !loaded {
		size.Add(1)
    } else {
		foundMap.Store(hex,found)
	}
	if size.Load() > 90000 {
		now:=time.Now().Unix()
		if last_clean+120 < now {
			last_clean=now
			go clean(s)
		}
	}
}

func (s *DataHandle) Load(hex string) (*postcodec.Post, bool) {
	val,ok:=foundMap.Load(hex)
	if ok {
		dat:=val.(*loadData)
		dat.last=time.Now().Unix()
		return dat.data,true
	}
	var v []byte
	var err error
	if is_redis{
		v, err = s.rdb.Get(s.ctx, hex).Bytes()
	} else {
		v, err = localget(hex)
	}
	if err != nil {
		return nil,false
	}
	if len(v)<4{
		return nil,false
	}
	p, err := postcodec.Decode(v)
    if err != nil {
		log.Println("postdb: decode err:",err)
        return nil,false
    }
	
	newfound:=&loadData{
		data: &p,
		last: time.Now().Unix(),
	}
	_,loaded := foundMap.LoadOrStore(hex,newfound)
    if !loaded {
		size.Add(1)
    } else {
		foundMap.Store(hex,newfound)
	}
	
	return newfound.data,true
}

func (s *DataHandle) Delete(hex string) {
	if _,ok := foundMap.LoadAndDelete(hex); ok {
		size.Add(-1)
	}
	var err error
	if is_redis{
		err = s.rdb.Del(s.ctx, hex).Err()
	} else {
		err = localdel(hex)
	}
	if err != nil {
		log.Println("postdb: del err:",err)
		return
	}
}

func clean(s *DataHandle){
	ok := lock.CompareAndSwap(false, true)
	if !ok {
		return
	}
	defer lock.Store(false)
	
	i:=0
	var keys []string
	now:=time.Now().Unix()-60*60*6
	foundMap.Range(func(k, v interface{}) bool {
		val:= v.(*loadData)
		if val.last<now{
			 key := k.(string)
			 keys = append(keys, key)
			 i++
		}
        return i<60000
    })
	if len(keys) == 0 {
		log.Println("load too high, clean fail")
        return
    }
	
	for _,hex:=range keys {
		if rnd.Intn(3) != 0 {
			if val,ok := foundMap.LoadAndDelete(hex); ok {
				push_redis(hex,val.(*loadData),s)
				size.Add(-1)
			}
		}
	}
}

func push_redis(hex string,dat *loadData,s *DataHandle){
	data, err := postcodec.Encode(dat.data)
	if err != nil {
		log.Println("postdb: encode err:",err)
		return
	}
	if is_redis {
		err = s.rdb.Set(s.ctx, hex, data, 0).Err()
	} else {
		err = localpush(hex,data)
	}
	if err != nil {
		log.Println("postdb: set-key err:",err)
		return
	}
}

func localget(hex string) ([]byte,error) {
	if len(hex)<4{
		return nil,os.ErrNotExist
	}
	openPath:=local_path+hex[:2]
	key,err:=strconv.ParseUint(hex[:2], 16, 8)
	if err!=nil{
		return nil,err
	}
	localLock[key].RLock()
	file,err:=os.Open(openPath)
	if err!=nil{
		localLock[key].RUnlock()
		return nil,err
	}
	defer func(){
		file.Close()
		localLock[key].RUnlock()
	}()
	mydb:=make(map[string][]byte)
	err = gob.NewDecoder(file).Decode(&mydb)
	if err !=nil {
		return nil,err
	}
	val,ok:=mydb[hex]
	if ok {
		return val,nil
	}
	return nil,os.ErrNotExist
}

func localpush(hex string,data []byte) error{
	if len(hex)<4{
		return os.ErrNotExist
	}
	openPath:=local_path+hex[:2]
	key,err:=strconv.ParseUint(hex[:2], 16, 8)
	if err!=nil{
		return err
	}
	localLock[key].Lock()
	file,err:=os.OpenFile(openPath, os.O_RDWR|os.O_CREATE, 0644)
	if err!=nil{
		localLock[key].Unlock()
		return err
	}
	defer func(){
		file.Close()
		localLock[key].Unlock()
	}()
	mydb:=make(map[string][]byte)
	err = gob.NewDecoder(file).Decode(&mydb)
	if err != nil && err != io.EOF {
		return err
	}
	file.Truncate(0)
	file.Seek(0,0)
	mydb[hex]=data
	return gob.NewEncoder(file).Encode(mydb)
}

func localdel(hex string) error {
	if len(hex)<4{
		return os.ErrNotExist
	}
	openPath:=local_path+hex[:2]
	key,err:=strconv.ParseUint(hex[:2], 16, 8)
	if err!=nil{
		return err
	}
	localLock[key].Lock()
	file,err:=os.OpenFile(openPath, os.O_RDWR|os.O_CREATE, 0644)
	if err!=nil{
		localLock[key].Unlock()
		return err
	}
	defer func(){
		file.Close()
		localLock[key].Unlock()
	}()
	mydb:=make(map[string][]byte)
	err = gob.NewDecoder(file).Decode(&mydb)
	if err !=nil {
		return err
	}
	file.Truncate(0)
	file.Seek(0,0)
	delete(mydb,hex)
	return gob.NewEncoder(file).Encode(mydb)
}

func fileTest(file string) bool {
	dir := filepath.Dir(file)
	info, err := os.Stat(dir)
	if err != nil {
		return false 
	}
	if !info.IsDir() {
		return false
	}
	temp, err := os.CreateTemp(dir, ".permcheck-*")
	if err != nil {
		return false
	}
	name := temp.Name()
	temp.Close()
	os.Remove(name)
	return true
}

func init(){
	log.SetLevel("info")
	rnd=rand.New(rand.NewSource(time.Now().UnixNano()))
}