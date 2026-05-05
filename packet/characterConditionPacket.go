package packet

import (
	"fmt"

	"github.com/Marcentus/Midir/util"
)

type CharacterConditionPacket struct {
	Id       uint64
	IsEnable bool
	EntityCharacterCondition
}

func ParseCharacterConditionPacket(p *GamePacket) (*CharacterConditionPacket, error) {
	if len(p.Msg) < 2 {
		return nil, fmt.Errorf("ParseCharacterConditionPacket: packet too short")
	}
	if p.Msg[0].Type() != MessageElemTypeByte {
		return nil, fmt.Errorf("ParseCharacterConditionPacket: isEnable has unexpected type %v", p.Msg[0].Type())
	}
	if p.Msg[1].Type() != MessageElemTypeInt {
		return nil, fmt.Errorf("ParseCharacterConditionPacket: ccId has unexpected type %v", p.Msg[1].Type())
	}

	isEnable := p.Msg[0].Data().(uint8) != 0
	ccId := p.Msg[1].Data().(uint32)

	if !isEnable {
		v := &CharacterConditionPacket{
			Id:       p.Id,
			IsEnable: false,
			EntityCharacterCondition: EntityCharacterCondition{
				CCId: ccId,
			},
		}

		return v, nil
	}

	if len(p.Msg) < 5 {
		return nil, fmt.Errorf("ParseCharacterConditionPacket: packet too short2")
	}

	if p.Msg[2].Type() != MessageElemTypeLong {
		return nil, fmt.Errorf("ParseCharacterConditionPacket: disableAt has unexpected type %v", p.Msg[2].Type())
	}
	if p.Msg[3].Type() != MessageElemTypeString {
		return nil, fmt.Errorf("ParseCharacterConditionPacket: metaData has unexpected type %v", p.Msg[3].Type())
	}
	if p.Msg[4].Type() != MessageElemTypeLong {
		return nil, fmt.Errorf("ParseCharacterConditionPacket: attackerId has unexpected type %v", p.Msg[4].Type())
	}

	disableAtRaw := p.Msg[2].Data().(uint64)
	metaData := p.Msg[3].Data().(string)
	attackerId := p.Msg[4].Data().(uint64)

	disableAt := util.ParseMabiTime(disableAtRaw).Unix()

	v := &CharacterConditionPacket{
		Id:       p.Id,
		IsEnable: true,
		EntityCharacterCondition: EntityCharacterCondition{
			CCId:       ccId,
			DisableAt:  disableAt,
			MetaData:   metaData,
			AttackerId: attackerId,
		},
	}

	return v, nil
}
