package postdb

import (
	"context"
	"github.com/redis/go-redis/v9"
	"os"
	"github.com/stalltrix/kepweb/postcodec"
	"github.com/stalltrix/kep-demo/logger"
	"github.com/stalltrix/kep-demo/kepdb"
	"github.com/stalltrix/kep-demo/kepresolv"
	"sync"
	"sync/atomic"
	"time"
	"math/rand"
	"path/filepath"
	"encoding/gob"
	"io"
	"strconv"
	"sort"
	"bytes"
	hx "encoding/hex"
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
	rebuildLock atomic.Bool
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
	var p postcodec.Post
	if err != nil || len(v)<4 {
		_,err=kepdb.FindFile(hex + ".txt")
		if err!=nil{
		return nil,false
		}
		pp,ok:=reBuildPost(hex)
		if !ok {
			return nil,false
		}
		p=*pp
	} else {
	p, err = postcodec.Decode(v)
    if err != nil {
		log.Println("postdb: decode err:",err)
        return nil,false
    }
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

func reBuildPost(tag string) (*postcodec.Post,bool) {
	ok := rebuildLock.CompareAndSwap(false, true)
	if !ok {
	//重建是情分
		return nil,false
	}
	defer rebuildLock.Store(false)
	hexs,err:=kepdb.ReadHash(tag)
	if err !=nil {
		return nil,false
	}
	dat,err:=kepresolv.Resolv(hexs)
	if err !=nil {
		return nil,false
	}
	if dat.Apoint_to != nil {
		return nil,false
	}
	txt:=dat.Atxt
	domain:=dat.Adomain
	timestamp:=dat.Atimestamp
	perm:=dat.Aperm
	key_des:=dat.Akey_des
	tag_i:=dat.Atag2
	var newRly = []postcodec.Reply{{ID: 1, User: string(domain), Meta: "", Me: true, Post: string(txt), Time: timestamp, Tag: tag_i, Hex: tag},}
	var lastID=2
	subs,err:=kepdb.ReadSub(tag)
	if err ==nil {
for _,sub := range subs {
	hex_byte,err:=kepdb.ReadHash(sub)
	if err ==nil {
	dat,err:=kepresolv.Resolv(hex_byte)
	if err !=nil {
	log.Println("load data err:",err)
	continue;
	}
 txt2:=dat.Atxt
 domain2:=dat.Adomain
 timestamp2:=dat.Atimestamp
 point_to2:=dat.Apoint_to
 perm2:=dat.Aperm
 key_des2:=dat.Akey_des
 tagi2:=dat.Atag2
	if tagi2 == 65534 {
		o_hex2:=hx.EncodeToString(point_to2)
		if o_hex2 == tag {
		if (key_des == key_des2) && bytes.Equal(domain,domain2) {
			if timestamp2 > newRly[0].Time{
			newRly[0]=postcodec.Reply{
    ID:   1,
    User: string(domain2),
    Meta: "",
    Me:   true,
    Post: string(txt2),
    Time: timestamp2,
	Tag: tag_i,
	Hex: tag,
			}
	perm=perm2
		}}
		}
		continue;
	}
 newRly = append(newRly, postcodec.Reply{
    ID:   lastID,
    User: string(domain2),
    Meta: "",
    Me:   (key_des == key_des2) && bytes.Equal(domain,domain2),
    Post: string(txt2),
    Time: timestamp2,
	Tag: tagi2,
	Hex: sub,
 })
 lastID++
	}
}
			}
	sort.Slice(newRly, func(i, j int) bool {
		if i==0||j==0{
			return false
		}
		return newRly[i].Time < newRly[j].Time
	})
	val:=&postcodec.Post{
        PostHex: tag,
        TagID:   tag_i,
        Owner:   string(domain),
        LastTime: timestamp,
        Replies: newRly,
		TypeID: perm,
    }
	return val,true
}

func init(){
	log.SetLevel("info")
	rnd=rand.New(rand.NewSource(time.Now().UnixNano()))
}