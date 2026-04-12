package mapvec

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stalltrix/kep-demo/logger"
	"github.com/stalltrix/kep-demo/pbb"
	"errors"
	"bytes"
)

type DataHandle struct {
	ctx context.Context
	rdb *redis.Client
}

type Map向量 struct {
    X string
	Y int
}

type cacheItem struct {
    val    [36]byte
    ori_key  [32]byte
}

var (
	log logger.Log_TYPE
	cache *pbb.Cache
	is_init bool
	is_redis bool
	now_handle *DataHandle
)

func New(ipaddr,passwd string,is_same bool) (*DataHandle, error) {
	if is_init {
		return nil,errors.New("init")
	}
	cache=pbb.NewMap() 
	if ipaddr==""{
		is_init=true
		return &DataHandle{},nil
	}
	dbName:=0
	if is_same {
		dbName=1
	}
	ctx := context.Background()
    rdb := redis.NewClient(&redis.Options{
        Addr:     ipaddr,
        Password: passwd,
        DB:       dbName,
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
	now_handle=st
	return st,nil
}

func Mux() (*DataHandle, error) {
	if !is_init {
		return nil,errors.New("!init")
	}
	return now_handle,nil
}

func (s *DataHandle) Store(hex string,dat Map向量) {
	var out [32]byte
	if !Hex64To32(&out, hex){
		log.Println("format input err:",hex)
		return
	}
	if len(dat.X)!=64{
		log.Println("input x err:",len(dat.X))
		return
	}
	内存数据,err:=To内存(dat)
	if err!=nil{
		log.Println("mapvec: to mem err:",err)
		return
	}
	newdata:=&cacheItem{
		val: 内存数据,
		ori_key: out,
	}
	cache.Store(string(out[:]),newdata)
	
	if !is_redis{
		return
	}

	databyte:=内存数据[:]
	
	need_renew:=false
	v, err := s.rdb.Get(s.ctx, hex).Bytes()
	if err != nil || len(v)!=36 {
		need_renew=true
	} else {
		if !bytes.Equal(databyte,v){
			need_renew=true
		}
	}
	
	if need_renew {
	err = s.rdb.Set(s.ctx, hex, databyte, 0).Err()
	if err != nil {
		log.Println("mapvec: set-key err:",err)
		return
	}
	}
}

func (s *DataHandle) Load(hex string) (Map向量, bool) {
	var out [32]byte
	if !Hex64To32(&out, hex){
		log.Println("format input err:",hex)
		return Map向量{},false
	}
	val,ok:=cache.Load(string(out[:]))
	if ok {
		dat:=val.(*cacheItem)
		if dat.ori_key==out{
			payload,err:=From内存(dat.val)
			if err!=nil{
				log.Println("mapvec: load",err)
			}
			return payload,true
		}
	}
	if !is_redis{
		return Map向量{},false
	}
	v, err := s.rdb.Get(s.ctx, hex).Bytes()
	
	if err != nil || len(v)!=36 {
		return Map向量{},false
	}
	
	var Mapdata [36]byte
	copy(Mapdata[:],v)
	
	payload,err:=From内存(Mapdata)
	if err!=nil{
		log.Println("mapvec: recv",err)
		return Map向量{},false
	}
	
	newdata:=&cacheItem{
		val: Mapdata,
		ori_key: out,
	}
	cache.Store(string(out[:]),newdata)
	
	return payload,true
}

func To内存(m Map向量) ([36]byte, error) {
    var out[36]byte
	var Xout[32]byte
	var Yout[4]byte
	
	if !Hex64To32(&Xout,m.X) {
		return out,errors.New("!hex64To32")
	}
	
	Yout[0]=byte(m.Y&255)
	Yout[1]=byte((m.Y>>8)&255)
	Yout[2]=byte((m.Y>>16)&255)
	Yout[3]=byte((m.Y>>24)&255)
	
	copy(out[:],Xout[:])
	copy(out[32:],Yout[:])
    return out, nil
}

func From内存(b [36]byte) (Map向量, error) {
    var m Map向量
	
	m.X = Bytes32ToHex64(b[:32])
	
	if m.X==""{
		return m,errors.New("decode x err")
	}

	var y int
	y=int(b[32])|int(b[33])<<8|int(b[34])<<16|int(b[35])<<24
    m.Y = y

    return m, nil
}

func Hex64To32(dst *[32]byte, s string) bool {
    if len(s) != 64 {
        return false
    }

    for i := 0; i < 32; i++ {
        hi := fromHex(s[i*2])
        lo := fromHex(s[i*2+1])
        if hi < 0 || lo < 0 {
            return false
        }
        dst[i] = byte(hi<<4 | lo)
    }
    return true
}

func Bytes32ToHex64(src []byte) string {
    const hex = "0123456789abcdef"
    out := make([]byte, 64)
	
	if len(src)!=32{
		return ""
	}

    for i := 0; i < 32; i++ {
        b := src[i]
        out[i*2] = hex[b>>4]
        out[i*2+1] = hex[b&0x0f]
    }

    return string(out)
}

func fromHex(c byte) int8 {
    switch {
    case '0' <= c && c <= '9':
        return int8(c - '0')
    case 'a' <= c && c <= 'f':
        return int8(c - 'a' + 10)
    case 'A' <= c && c <= 'F':
        return int8(c - 'A' + 10)
    }
    return -1
}

func init(){
	log.SetLevel("info")
}