package main

import (
	"context"
	"fmt"
	"golang.org/x/time/rate"
	"math"
	"math/rand"
	"sync"
	"time"
)

const RTTDATASET = 100  //rtt数据集
const SAMPLESET  = 90   //样本数据集
const MODULUS    = 0.8  //srtt经典算法更新系数

const UBOUND = 800.0     //rto超时上限
const LBOUND = 400.0     //rto下限
const RTOMODULUS  = 1.5  //rto变化系数

type ALimit struct {
	rttset    []float64
	cwnd      rate.Limit
	sshthresh rate.Limit
	srtt      float64
	rto       float64
	mutex     sync.Mutex
	l         *rate.Limiter
}

func NewALimit() *ALimit{
	return &ALimit{
		rttset:make([]float64,0,RTTDATASET),
		cwnd:1,
		sshthresh:30000,
		srtt:0,
		rto:LBOUND,
		l:rate.NewLimiter(1,1),
	}
}

//收集rtt
func (al *ALimit)addRtt(rtt float64)  {

	al.mutex.Lock()
	defer al.mutex.Unlock()

	rto       := al.rto
	cwnd      := al.cwnd
	sshthresh := al.sshthresh
	srtt      := al.srtt

	if rtt < rto{
		cwnd = al.updateCWND(cwnd,sshthresh)
	}else {
		cwnd, sshthresh = al.fastRecovery(cwnd,sshthresh)
	}

	al.rttset = append(al.rttset, rtt)
	n := len(al.rttset)

	if n >= RTTDATASET {
		spg := make([]float64,SAMPLESET,SAMPLESET)
		al.sampling(spg, al.rttset)

		rtt := al.averageRTT(spg)
		srtt = al.updateSRTT(srtt, rtt)

		if srtt >= rto {
			cwnd, sshthresh = al.blocking(cwnd,sshthresh)
		}
		rto = al.updateRTO(srtt)
		al.rttset = al.rttset[:0]
	}

	//更新数据
	al.rto = rto
	al.cwnd = cwnd
	al.sshthresh = sshthresh
	al.srtt = srtt

}

//采样蓄水池算法
func (al *ALimit)sampling(spg []float64, rttset []float64) {

	k := SAMPLESET
	n := len(rttset)

	for i:=0;i< k;i++  {
		spg[i] = rttset[i]
	}

	for i:=len(spg);i< n;i++ {
		r := rand.Intn(i)
		if r < k {
			spg[r] = rttset[i]
		}
	}
}

//平均rtt值
func (al *ALimit)averageRTT(samples []float64) float64{
	total := 0.0
	for _,s := range samples{
		total += s
	}
	averageRtt := total/ float64(len(samples))
	return averageRtt
}

//经典算法
func (alim *ALimit)updateSRTT(srtt float64, rtt float64) float64{
	srtt = srtt * MODULUS + (1-MODULUS) * rtt
	return srtt
}

//更新rto
func (alim *ALimit)updateRTO(srtt float64) float64{
	rto := math.Min(UBOUND,math.Max(srtt * RTOMODULUS,LBOUND))
	return rto
}

func (alim *ALimit)updateCWND(cwnd rate.Limit, sshthresh rate.Limit) rate.Limit{
	if cwnd < sshthresh {
		//慢启动
		return alim.slowStart(cwnd)
	}else {
		//避免拥塞
		return alim.avoidBlocking(cwnd)
	}
}
//慢启动
func (alim *ALimit)slowStart(cwnd rate.Limit) rate.Limit{
	cwnd = cwnd * 2
	alim.l.SetLimit(cwnd)
	return cwnd
}
//拥塞避免
func (alim *ALimit)avoidBlocking(cwnd rate.Limit) rate.Limit{
	cwnd = cwnd + 1
	alim.l.SetLimit(cwnd)
	return cwnd
}

//快速恢复
func (alim *ALimit)fastRecovery(cwnd rate.Limit, sshthresh rate.Limit) (rate.Limit, rate.Limit){

	// +1保证最差情况速率从1开始
	if cwnd >= sshthresh {
		// sshthresh 拥塞避免情况快速恢复
		cwnd = sshthresh + 1
	}else {
		// 慢启动 情况下快速恢复
		cwnd = cwnd/2 + 1
	}

	alim.l.SetLimit(cwnd)

	fmt.Println("快速恢复：","sshthresh = ",sshthresh,"cwnd = ",cwnd)
	x = 1

	return cwnd, sshthresh

}

//拥塞阻塞
func (al *ALimit)blocking(cwnd rate.Limit, sshthresh rate.Limit) (rate.Limit, rate.Limit){

	if cwnd <= sshthresh {
		sshthresh = cwnd/2
	}
	cwnd = 1
	//更新拥塞窗口
	al.l.SetLimit(cwnd)


	x = 1
	//fmt.Println("============拥塞阻塞===================")
	//fmt.Println("srtt = ",al.srtt)
	//fmt.Println("rto = ",al.rto)
	//fmt.Println("cwnd = ",al.cwnd)
	//fmt.Println("sshthresh = ",al.sshthresh)

	return cwnd, sshthresh

}

//rtt计算
func (al *ALimit)CalculateRtt(t1 time.Time){
	now := time.Now()
	elapsed := now.Sub(t1).Seconds() * 1000

	//fmt.Println("===============================")
	//fmt.Println("rtt = ",elapsed)
	//fmt.Println("srtt = ",al.srtt)
	//fmt.Println("rto = ",al.rto)
	//fmt.Println("cwnd = ",al.cwnd)
	//fmt.Println("sshthresh = ",al.sshthresh)

	al.addRtt(elapsed)
}

//阻塞
func (alim *ALimit)WaitL(ctx context.Context) (err error){
	return alim.l.Wait(ctx)
}

var x = 1
func main() {
	alimt := NewALimit()
	c := context.TODO()
	for true {
		alimt.WaitL(c)
		go logic(alimt)

		if x > 800 {
			x = 1

			//time.Sleep(time.Hour)
		}else {
			x ++
		}
		fmt.Println(x)
	}
}

func logic(l *ALimit)  {
	t1 := time.Now()
	//模拟逻辑处理
	BusinessLogic()

	l.CalculateRtt(t1)

}

func BusinessLogic()  {
	//模拟业务逻辑，线性增加
	time.Sleep(time.Second/10 + time.Duration(1e6*x))
}