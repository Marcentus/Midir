package main

import (
	"sync"
	"time"

	"github.com/Marcentus/Midir/packet"
)

type entityCache map[uint64]*entityInfoExtend

type entityInfoExtend struct {
	*packet.EntityInfo
	sync.Mutex
	appearAt              int64
	disappearAt           int64
	characterConditionMap map[uint32]*packet.EntityCharacterCondition
	equipItemMap          map[uint32]*packet.EntityItem
}

func (t entityCache) add(p *packet.EntityInfo, at time.Time) {
	if e, ok := t[p.Id]; ok {
		e.EntityInfo = p
		e.appearAt = at.Unix()
		e.disappearAt = 0
		return
	}

	t[p.Id] = &entityInfoExtend{
		EntityInfo:            p,
		appearAt:              at.Unix(),
		disappearAt:           0,
		characterConditionMap: make(map[uint32]*packet.EntityCharacterCondition),
		equipItemMap:          make(map[uint32]*packet.EntityItem),
	}

	t.cleanup(at)
}

func (t entityCache) disappear(id uint64, at time.Time) {
	e := t[id]
	if e == nil {
		return
	}

	e.disappearAt = at.Unix()
}

func (t entityCache) addCondition(p *packet.CharacterConditionPacket) {
	e := t[p.Id]
	if e == nil {
		return
	}

	if !p.IsEnable {
		e.Lock()
		delete(e.characterConditionMap, p.CCId)
		e.Unlock()
		return
	}

	e.Lock()
	e.characterConditionMap[p.CCId] = &p.EntityCharacterCondition
	e.Unlock()

	/*
		go func() {
			time.Sleep(time.Until(time.Unix(p.DisableAt, 0)))
			e.Lock()
			if cond := e.characterConditionMap[p.CCId]; cond != nil && cond.DisableAt == p.DisableAt {
				delete(e.characterConditionMap, p.CCId)
			}
			e.Unlock()
		}()
	*/
}

// return -> isUpdated
func (t entityCache) addOrUpdateCondition(id uint64, p *packet.EntityCharacterCondition) bool {
	e := t[id]
	if e == nil {
		return false
	}

	e.Lock()
	defer e.Unlock()

	/*
		setDisableTimer := func() {
			time.Sleep(time.Until(time.Unix(p.DisableAt, 0)))
			e.Lock()
			if cond := e.characterConditionMap[p.CCId]; cond != nil && cond.DisableAt == p.DisableAt {
				delete(e.characterConditionMap, p.CCId)
			}
			e.Unlock()
		}
	*/

	if cond := e.characterConditionMap[p.CCId]; cond != nil {
		isSame := *cond == *p
		if isSame {
			return false
		}

		*cond = *p
		// go setDisableTimer()
		return true
	}

	e.characterConditionMap[p.CCId] = p
	// go setDisableTimer()
	return true
}

func (t entityCache) addOrUpdateEquipItem(id uint64, p *packet.EntityItem) bool {
	e := t[id]
	if e == nil {
		return false
	}

	e.Lock()
	defer e.Unlock()

	if item := e.equipItemMap[p.PocketType]; item != nil {
		isSame := *item == *p
		if isSame {
			return false
		}

		*item = *p
		return true
	}

	e.equipItemMap[p.PocketType] = p
	return true
}

func (t entityCache) hasEquipItem(id uint64, pocketType uint32) bool {
	e := t[id]
	if e == nil {
		return false
	}

	e.Lock()
	defer e.Unlock()

	if item := e.equipItemMap[pocketType]; item != nil {
		return true
	}

	return false
}

func (t entityCache) allEquipItemPockets(id uint64) []uint32 {
	e := t[id]
	if e == nil {
		return nil
	}

	e.Lock()
	defer e.Unlock()

	var pockets []uint32
	for k := range e.equipItemMap {
		pockets = append(pockets, k)
	}

	return pockets
}

func (t entityCache) unequipItem(id uint64, pocketType uint32) {
	e := t[id]
	if e == nil {
		return
	}

	e.Lock()
	defer e.Unlock()

	delete(e.equipItemMap, pocketType)
}

func (t entityCache) updateBody(id uint64, height float32, weight float32, upper float32, lower float32) {
	e := t[id]
	if e == nil {
		return
	}

	e.Lock()
	defer e.Unlock()

	e.Height = height
	e.Weight = weight
	e.Upper = upper
	e.Lower = lower
}

func (t entityCache) cleanup(at time.Time) {
	now := at.Unix()
	mobRemoveSec, userRemoveSec := int64(1*60), int64(5*60)
	noDisappearMobRemoveSec, noDisappearUserRemoveSec := int64(12*60*60), int64(3*60*60)

	for k, v := range t {
		if v.disappearAt == 0 {
			if v.IsUser() {
				if now-v.appearAt > noDisappearUserRemoveSec {
					delete(t, k)
				}
			}

			if now-v.appearAt > noDisappearMobRemoveSec {
				delete(t, k)
			}

			continue
		}

		if v.IsUser() {
			if now-v.disappearAt > userRemoveSec {
				delete(t, k)
			}
			continue
		}

		if now-v.disappearAt > mobRemoveSec {
			delete(t, k)
		}
	}
}

func (t *entityInfoExtend) IsUser() bool {
	switch t.RaceId {
	case 8001, 8002, 9001, 9002, 10001, 10002:
		return true
	}

	return false
}
