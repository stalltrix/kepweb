package randlist

import (
	"time"
	"math/rand"
	"sync/atomic"
)

var (
	rnd *rand.Rand
	randList [256]string
	randTime int64
	loadHex int
	lock atomic.Bool
)

func init(){
	rnd=rand.New(rand.NewSource(time.Now().UnixNano()))
}

func Renew(lastList *[65536]string,lastIdx uint16){
	ok := lock.CompareAndSwap(false, true)
	if !ok {
		return
	}
	defer lock.Store(false)
	nowTime:=time.Now().Unix()
	if randTime + 3600 > nowTime {
		return
	}
	randTime=nowTime
	i:=lastIdx
	k:=0
	diff :=make(map[string]struct{})
	for j:=0;j<512;j++{
		i--
		hex:=lastList[i]
		if hex == "" {
			break
		}
		_,ok:=diff[hex]
		if ok {
			continue
		}
		randList[k]=hex
		diff[hex]=struct{}{}
		k++
		if k > 255 {
			break
		}
	}
	loadHex=k
	for i:=k-1;i>0;i-- {
        j := rnd.Intn(i+1)
        randList[i], randList[j] = randList[j], randList[i]
    }
}

func MustRenew() bool {
	return randTime + 3600 < time.Now().Unix()
}

func GetList() (*[256]string){
	return &randList
}