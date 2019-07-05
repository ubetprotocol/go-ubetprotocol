// Copyright 2016 The go-ubetprotocol Authors
// This file is part of the go-ubetprotocol library.
//
// The go-ubetprotocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ubetprotocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more detailct.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ubetprotocol library. If not, see <http://www.gnu.org/licenses/>.

package les

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ubetprotocol/go-ubetprotocol/common/mclock"
	"github.com/ubetprotocol/go-ubetprotocol/eth"
	"github.com/ubetprotocol/go-ubetprotocol/ethdb"
	"github.com/ubetprotocol/go-ubetprotocol/les/csvlogger"
	"github.com/ubetprotocol/go-ubetprotocol/les/flowcontrol"
	"github.com/ubetprotocol/go-ubetprotocol/log"
)

const makeCostStats = false // make request cost statistics during operation

var (
	// average request cost estimates based on serving time
	reqAvgTimeCost = requestCostTable{
		GetBlockHeadersMsg:     {150000, 30000},
		GetBlockBodiesMsg:      {0, 700000},
		GetReceiptsMsg:         {0, 1000000},
		GetCodeMsg:             {0, 450000},
		GetProofsV2Msg:         {0, 600000},
		GetHelperTrieProofsMsg: {0, 1000000},
		SendTxV2Msg:            {0, 450000},
		GetTxStatusMsg:         {0, 250000},
	}
	// maximum incoming message size estimates
	reqMaxInSize = requestCostTable{
		GetBlockHeadersMsg:     {40, 0},
		GetBlockBodiesMsg:      {0, 40},
		GetReceiptsMsg:         {0, 40},
		GetCodeMsg:             {0, 80},
		GetProofsV2Msg:         {0, 80},
		GetHelperTrieProofsMsg: {0, 20},
		SendTxV2Msg:            {0, 16500},
		GetTxStatusMsg:         {0, 50},
	}
	// maximum outgoing message size estimates
	reqMaxOutSize = requestCostTable{
		GetBlockHeadersMsg:     {0, 556},
		GetBlockBodiesMsg:      {0, 100000},
		GetReceiptsMsg:         {0, 200000},
		GetCodeMsg:             {0, 50000},
		GetProofsV2Msg:         {0, 4000},
		GetHelperTrieProofsMsg: {0, 4000},
		SendTxV2Msg:            {0, 100},
		GetTxStatusMsg:         {0, 100},
	}
	// request amounts that have to fit into the minimum buffer size minBufferMultiplier times
	minBufferReqAmount = map[uint64]uint64{
		GetBlockHeadersMsg:     192,
		GetBlockBodiesMsg:      1,
		GetReceiptsMsg:         1,
		GetCodeMsg:             1,
		GetProofsV2Msg:         1,
		GetHelperTrieProofsMsg: 16,
		SendTxV2Msg:            8,
		GetTxStatusMsg:         64,
	}
	minBufferMultiplier = 3
)

const (
	maxCostFactor    = 2 // ratio of maximum and average cost estimates
	gfUsageThreshold = 0.5
	gfUsageTC        = time.Second
	gfRaiseTC        = time.Second * 200
	gfDropTC         = time.Second * 50
	gfDbKey          = "_globalCostFactorV3"
)

// costTracker is responsible for calculating costs and cost estimates on the
// server side. It continuously updates the global cost factor which is defined
// as the number of cost units per nanosecond of serving time in a single thread.
// It is based on statistics collected during serving requests in high-load periods
// and practically acts as a one-dimension request price scaling factor over the
// pre-defined cost estimate table. Instead of scaling the cost values, the real
// value of cost units is changed by applying the factor to the serving times. This
// is more convenient because the changes in the cost factor can be applied immediately
// without always notifying the clients about the changed cost tables.
type costTracker struct {
	db     ethdb.Database
	stopCh chan chan struct{}

	inSizeFactor, outSizeFactor float64
	gf, utilTarget              float64
	minBufLimit                 uint64

	gfUpdateCh      chan gfUpdate
	gfLock          sync.RWMutex
	totalRechargeCh chan uint64

	stats                                                     map[uint64][]uint64
	logger                                                    *csvlogger.Logger
	logRecentTime, logRecentAvg, logTotalRecharge, logRelCost *csvlogger.Channel
}

// newCostTracker creates a cost tracker and loads the cost factor statistics from the database.
// It also returns the minimum capacity that can be assigned to any peer.
func newCostTracker(db ethdb.Database, config *eth.Config, logger *csvlogger.Logger) (*costTracker, uint64) {
	utilTarget := float64(config.LightServ) * flowcontrol.FixedPointMultiplier / 100
	ct := &costTracker{
		db:               db,
		stopCh:           make(chan chan struct{}),
		utilTarget:       utilTarget,
		logger:           logger,
		logRelCost:       logger.NewMinMaxChannel("relativeCost", true),
		logRecentTime:    logger.NewMinMaxChannel("recentTime", true),
		logRecentAvg:     logger.NewMinMaxChannel("recentAvg", true),
		logTotalRecharge: logger.NewChannel("totalRecharge", 0.01),
	}
	if config.LightBandwidthIn > 0 {
		ct.inSizeFactor = utilTarget / float64(config.LightBandwidthIn)
	}
	if config.LightBandwidthOut > 0 {
		ct.outSizeFactor = utilTarget / float64(config.LightBandwidthOut)
	}
	if makeCostStats {
		ct.stats = make(map[uint64][]uint64)
		for code := range reqAvgTimeCost {
			ct.stats[code] = make([]uint64, 10)
		}
	}
	ct.gfLoop()
	costList := ct.makeCostList(ct.globalFactor() * 1.25)
	for _, c := range costList {
		amount := minBufferReqAmount[c.MsgCode]
		cost := c.BaseCost + amount*c.ReqCost
		if cost > ct.minBufLimit {
			ct.minBufLimit = cost
		}
	}
	ct.minBufLimit *= uint64(minBufferMultiplier)
	return ct, (ct.minBufLimit-1)/bufLimitRatio + 1
}

// stop stops the cost tracker and saves the cost factor statistics to the database
func (ct *costTracker) stop() {
	stopCh := make(chan struct{})
	ct.stopCh <- stopCh
	<-stopCh
	if makeCostStats {
		ct.printStats()
	}
}

// makeCostList returns upper cost estimates based on the hardcoded cost estimate
// tables and the optionally specified incoming/outgoing bandwidth limits
func (ct *costTracker) makeCostList(globalFactor float64) RequestCostList {
	maxCost := func(avgTimeCost, inSize, outSize uint64) uint64 {
		cost := avgTimeCost * maxCostFactor
		inSizeCost := uint64(float64(inSize) * ct.inSizeFactor * globalFactor)
		if inSizeCost > cost {
			cost = inSizeCost
		}
		outSizeCost := uint64(float64(outSize) * ct.outSizeFactor * globalFactor)
		if outSizeCost > cost {
			cost = outSizeCost
		}
		return cost
	}
	var list RequestCostList
	for code, data := range reqAvgTimeCost {
		baseCost := maxCost(data.baseCost, reqMaxInSize[code].baseCost, reqMaxOutSize[code].baseCost)
		reqCost := maxCost(data.reqCost, reqMaxInSize[code].reqCost, reqMaxOutSize[code].reqCost)
		if ct.minBufLimit != 0 {
			// if minBufLimit is set then always enforce maximum request cost <= minBufLimit
			maxCost := baseCost + reqCost*minBufferReqAmount[code]
			if maxCost > ct.minBufLimit {
				mul := 0.999 * float64(ct.minBufLimit) / float64(maxCost)
				baseCost = uint64(float64(baseCost) * mul)
				reqCost = uint64(float64(reqCost) * mul)
			}
		}

		list = append(list, requestCostListItem{
			MsgCode:  code,
			BaseCost: baseCost,
			ReqCost:  reqCost,
		})
	}
	return list
}

type gfUpdate struct {
	avgTimeCost, servingTime float64
}

// gfLoop starts an event loop which updates the global cost factor which is
// calculated as a weighted average of the average estimate / serving time ratio.
// The applied weight equals the serving time if gfUsage is over a threshold,
// zero otherwise. gfUsage is the recent average serving time per time unit in
// an exponential moving window. This ensures that statistics are collected only
// under high-load circumstances where the measured serving times are relevant.
// The total recharge parameter of the flow control system which controls the
// total allowed serving time per second but nominated in cost units, should
// also be scaled with the cost factor and is also updated by this loop.
func (ct *costTracker) gfLoop() {
	var gfLog, recentTime, recentAvg float64
	lastUpdate := mclock.Now()
	expUpdate := lastUpdate

	data, _ := ct.db.Get([]byte(gfDbKey))
	if len(data) == 8 {
		gfLog = math.Float64frombits(binary.BigEndian.Uint64(data[:]))
	}
	gf := math.Exp(gfLog)
	ct.gf = gf
	totalRecharge := ct.utilTarget * gf
	ct.gfUpdateCh = make(chan gfUpdate, 100)
	threshold := gfUsageThreshold * float64(gfUsageTC) * ct.utilTarget / 1000000

	go func() {
		saveCostFactor := func() {
			var data [8]byte
			binary.BigEndian.PutUint64(data[:], math.Float64bits(gfLog))
			ct.db.Put([]byte(gfDbKey), data[:])
			log.Debug("global cost factor saved", "value", gf)
		}
		saveTicker := time.NewTicker(time.Minute * 10)

		for {
			select {
			case r := <-ct.gfUpdateCh:
				now := mclock.Now()
				if ct.logRelCost != nil && r.avgTimeCost > 1e-20 {
					ct.logRelCost.Update(r.servingTime * gf / r.avgTimeCost)
				}
				if r.servingTime > 1000000000 {
					ct.logger.Event(fmt.Sprintf("Very long servingTime = %f  avgTimeCost = %f  costFactor = %f", r.servingTime, r.avgTimeCost, gf))
				}
				dt := float64(now - expUpdate)
				expUpdate = now
				exp := math.Exp(-dt / float64(gfUsageTC))
				// calculate gf correction until now, based on previous values
				var gfCorr float64
				max := recentTime
				if recentAvg > max {
					max = recentAvg
				}
				// we apply continuous correction when MAX(recentTime, recentAvg) > threshold
				if max > threshold {
					// calculate correction time between last expUpdate and now
					if max*exp >= threshold {
						gfCorr = dt
					} else {
						gfCorr = math.Log(max/threshold) * float64(gfUsageTC)
					}
					// calculate log(gf) correction with the right direction and time constant
					if recentTime > recentAvg {
						// drop gf if actual serving times are larger than average estimates
						gfCorr /= -float64(gfDropTC)
					} else {
						// raise gf if actual serving times are smaller than average estimates
						gfCorr /= float64(gfRaiseTC)
					}
				}
				// update recent cost values with current request
				recentTime = recentTime*exp + r.servingTime
				recentAvg = recentAvg*exp + r.avgTimeCost/gf

				if gfCorr != 0 {
					gfLog += gfCorr
					gf = math.Exp(gfLog)
					if time.Duration(now-lastUpdate) > time.Second {
						totalRecharge = ct.utilTarget * gf
						lastUpdate = now
						ct.gfLock.Lock()
						ct.gf = gf
						ch := ct.totalRechargeCh
						ct.gfLock.Unlock()
						if ch != nil {
							select {
							case ct.totalRechargeCh <- uint64(totalRecharge):
							default:
							}
						}
						log.Debug("global cost factor updated", "gf", gf)
					}
				}
				ct.logRecentTime.Update(recentTime)
				ct.logRecentAvg.Update(recentAvg)
				ct.logTotalRecharge.Update(totalRecharge)

			case <-saveTicker.C:
				saveCostFactor()

			case stopCh := <-ct.stopCh:
				saveCostFactor()
				close(stopCh)
				return
			}
		}
	}()
}

// globalFactor returns the current value of the global cost factor
func (ct *costTracker) globalFactor() float64 {
	ct.gfLock.RLock()
	defer ct.gfLock.RUnlock()

	return ct.gf
}

// totalRecharge returns the current total recharge parameter which is used by
// flowcontrol.ClientManager and is scaled by the global cost factor
func (ct *costTracker) totalRecharge() uint64 {
	ct.gfLock.RLock()
	defer ct.gfLock.RUnlock()

	return uint64(ct.gf * ct.utilTarget)
}

// subscribeTotalRecharge returns all future updates to the total recharge value
// through a channel and also returns the current value
func (ct *costTracker) subscribeTotalRecharge(ch chan uint64) uint64 {
	ct.gfLock.Lock()
	defer ct.gfLock.Unlock()

	ct.totalRechargeCh = ch
	return uint64(ct.gf * ct.utilTarget)
}

// updateStats updates the global cost factor and (if enabled) the real cost vs.
// average estimate statistics
func (ct *costTracker) updateStats(code, amount, servingTime, realCost uint64) {
	avg := reqAvgTimeCost[code]
	avgTimeCost := avg.baseCost + amount*avg.reqCost
	select {
	case ct.gfUpdateCh <- gfUpdate{float64(avgTimeCost), float64(servingTime)}:
	default:
	}
	if makeCostStats {
		realCost <<= 4
		l := 0
		for l < 9 && realCost > avgTimeCost {
			l++
			realCost >>= 1
		}
		atomic.AddUint64(&ct.stats[code][l], 1)
	}
}

// realCost calculates the final cost of a request based on actual serving time,
// incoming and outgoing message size
//
// Note: message size is only taken into account if bandwidth limitation is applied
// and the cost based on either message size is greater than the cost based on
// serving time. A maximum of the three costs is applied instead of their sum
// because the three limited resources (serving thread time and i/o bandwidth) can
// also be maxed out simultaneously.
func (ct *costTracker) realCost(servingTime uint64, inSize, outSize uint32) uint64 {
	cost := float64(servingTime)
	inSizeCost := float64(inSize) * ct.inSizeFactor
	if inSizeCost > cost {
		cost = inSizeCost
	}
	outSizeCost := float64(outSize) * ct.outSizeFactor
	if outSizeCost > cost {
		cost = outSizeCost
	}
	return uint64(cost * ct.globalFactor())
}

// printStats prints the distribution of real request cost relative to the average estimates
func (ct *costTracker) printStats() {
	if ct.stats == nil {
		return
	}
	for code, arr := range ct.stats {
		log.Info("Request cost statistics", "code", code, "1/16", arr[0], "1/8", arr[1], "1/4", arr[2], "1/2", arr[3], "1", arr[4], "2", arr[5], "4", arr[6], "8", arr[7], "16", arr[8], ">16", arr[9])
	}
}

type (
	// requestCostTable assigns a cost estimate function to each request type
	// which is a linear function of the requested amount
	// (cost = baseCost + reqCost * amount)
	requestCostTable map[uint64]*requestCosts
	requestCosts     struct {
		baseCost, reqCost uint64
	}

	// RequestCostList is a list representation of request costs which is used for
	// database storage and communication through the network
	RequestCostList     []requestCostListItem
	requestCostListItem struct {
		MsgCode, BaseCost, ReqCost uint64
	}
)

// getMaxCost calculates the estimated cost for a given request type and amount
func (table requestCostTable) getMaxCost(code, amount uint64) uint64 {
	costs := table[code]
	return costs.baseCost + amount*costs.reqCost
}

// decode converts a cost list to a cost table
func (list RequestCostList) decode(protocolLength uint64) requestCostTable {
	table := make(requestCostTable)
	for _, e := range list {
		if e.MsgCode < protocolLength {
			table[e.MsgCode] = &requestCosts{
				baseCost: e.BaseCost,
				reqCost:  e.ReqCost,
			}
		}
	}
	return table
}

// testCostList returns a dummy request cost list used by tests
func testCostList(testCost uint64) RequestCostList {
	cl := make(RequestCostList, len(reqAvgTimeCost))
	var max uint64
	for code := range reqAvgTimeCost {
		if code > max {
			max = code
		}
	}
	i := 0
	for code := uint64(0); code <= max; code++ {
		if _, ok := reqAvgTimeCost[code]; ok {
			cl[i].MsgCode = code
			cl[i].BaseCost = testCost
			cl[i].ReqCost = 0
			i++
		}
	}
	return cl
}