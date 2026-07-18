package packet

import (
	"bytes"
	"fmt"

	"github.com/Marcentus/Midir/util"
)

type EntityInfo struct {
	Id                    uint64
	Name                  string
	RaceId                uint32
	SkinColor             uint8
	EyeType               uint16
	LeftEyeColor          uint8
	RightEyeColor         uint8
	MouthType             uint16
	Height                float32
	Weight                float32
	Upper                 float32
	Lower                 float32
	TitleId               uint32
	SubTitleId            uint32
	StyleTitleId          uint32
	StyleSubTitleId       uint32
	EquipItemMap          map[uint32]*EntityItem
	CharacterConditionMap map[uint32]*EntityCharacterCondition
	GuildName             string
	OwnerId               uint64 // 펫, 마리오네트 등

	CombatPower       float32 // From element[26]
	CurrentLevel      uint16  // From element[30]
	TotalLevel        uint32  // From element[31]
	Age               uint16  // From element[32]
	CurrentHP         float32 // From element[33]
	BaseHP            float32 // From element[34]
	AdditionalHP      float32 // From element[35]
	OverflowMaxHP     float32 // From element[36]
	CurrentVitalSurge float32 // From element[37]
	MaxVitalSurge     float32 // From element[38]
	MaxHP             float32 // Calculated as BaseHP + AdditionalHP
}

type EntityItem struct {
	// public data
	PocketType uint32
	ItemId     uint32
	Color1     uint32
	Color2     uint32
	Color3     uint32
	Color4     uint32
	Color5     uint32
	Color6     uint32
	Color7     uint32
	Amount     uint16
	UniqueId   uint64 // Added UniqueId
}

type EntityCharacterCondition struct {
	CCId       uint32
	DisableAt  int64
	MetaData   string
	AttackerId uint64
}

func ParseEntityAppearPacket(msg Message) (*EntityInfo, error) {
	origMsg := msg

	curPos := func() int {
		return len(origMsg) - len(msg)
	}

	if len(msg) < 2 || msg[1].Type() != MessageElemTypeByte {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		// logger.Println(err) // Spams console
		return nil, err
	}

	if msg[1].Data().(uint8) != 5 {
		// public 데이터만 읽음
		return nil, nil
	}

	v := &EntityInfo{
		EquipItemMap:          make(map[uint32]*EntityItem),
		CharacterConditionMap: make(map[uint32]*EntityCharacterCondition),
	}

	if len(msg) < 50 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		return nil, err
	}

	if msg[0].Type() != MessageElemTypeLong {
		err := fmt.Errorf("id has unexpected type %v", msg[0].Type())
		logger.Println(err)
		return nil, err
	}

	v.Id = msg[0].Data().(uint64)

	if msg[2].Type() != MessageElemTypeString {
		err := fmt.Errorf("name has unexpected type %v", msg[2].Type())
		logger.Println(err)
		return nil, err
	}

	v.Name = msg[2].Data().(string)


	if msg[5].Type() != MessageElemTypeInt {
		err := fmt.Errorf("raceId has unexpected type %v", msg[5].Type())
		logger.Println(err)
		return nil, err
	}

	v.RaceId = msg[5].Data().(uint32)

	if msg[6].Type() != MessageElemTypeByte {
		err := fmt.Errorf("skinColor has unexpected type %v", msg[6].Type())
		logger.Println(err)
		return nil, err
	}

	v.SkinColor = msg[6].Data().(uint8)

	if msg[7].Type() != MessageElemTypeShort {
		err := fmt.Errorf("eyeType has unexpected type %v", msg[7].Type())
		logger.Println(err)
		return nil, err
	}

	v.EyeType = msg[7].Data().(uint16)

	if msg[8].Type() != MessageElemTypeByte {
		err := fmt.Errorf("eyeColor has unexpected type %v", msg[8].Type())
		logger.Println(err)
		return nil, err
	}

	eyeColor := msg[8].Data().(uint8)

	if msg[9].Type() != MessageElemTypeShort {
		err := fmt.Errorf("mouthType has unexpected type %v", msg[9].Type())
		logger.Println(err)
		return nil, err
	}

	v.MouthType = msg[9].Data().(uint16)

	if msg[13].Type() != MessageElemTypeFloat {
		err := fmt.Errorf("height has unexpected type %v", msg[13].Type())
		logger.Println(err)
		return nil, err
	}

	v.Height = msg[13].Data().(float32)

	if msg[14].Type() != MessageElemTypeFloat {
		err := fmt.Errorf("weight has unexpected type %v", msg[14].Type())
		logger.Println(err)
		return nil, err
	}

	v.Weight = msg[14].Data().(float32)

	if msg[15].Type() != MessageElemTypeFloat {
		err := fmt.Errorf("upper has unexpected type %v", msg[15].Type())
		logger.Println(err)
		return nil, err
	}

	v.Upper = msg[15].Data().(float32)

	if msg[16].Type() != MessageElemTypeFloat {
		err := fmt.Errorf("lower has unexpected type %v", msg[16].Type())
		logger.Println(err)
		return nil, err
	}

	v.Lower = msg[16].Data().(float32)

	if msg[26].Type() == MessageElemTypeFloat {
		v.CombatPower = msg[26].Data().(float32)
	}
	if msg[30].Type() == MessageElemTypeShort {
		v.CurrentLevel = msg[30].Data().(uint16)
	}
	if msg[31].Type() == MessageElemTypeInt {
		v.TotalLevel = msg[31].Data().(uint32) + uint32(v.CurrentLevel)
	}
	if msg[32].Type() == MessageElemTypeShort {
		v.Age = msg[32].Data().(uint16)
	}
	if msg[33].Type() == MessageElemTypeFloat {
		v.CurrentHP = msg[33].Data().(float32)
	}
	if msg[34].Type() == MessageElemTypeFloat {
		v.BaseHP = msg[34].Data().(float32)
	}
	if msg[35].Type() == MessageElemTypeFloat {
		v.AdditionalHP = msg[35].Data().(float32)
	}
	if msg[36].Type() == MessageElemTypeFloat {
		v.OverflowMaxHP = msg[36].Data().(float32)
	}
	if msg[37].Type() == MessageElemTypeFloat {
		v.CurrentVitalSurge = msg[37].Data().(float32)
	}
	if len(msg) > 38 && msg[38].Type() == MessageElemTypeFloat {
		v.MaxVitalSurge = msg[38].Data().(float32)
	}
	v.MaxHP = v.BaseHP + v.AdditionalHP

	if msg[28].Type() != MessageElemTypeByte {
		err := fmt.Errorf("leftEyeColor has unexpected type %v", msg[28].Type())
		logger.Println(err)
		return nil, err
	}

	v.LeftEyeColor = msg[28].Data().(uint8)

	if v.LeftEyeColor == 0 {
		v.LeftEyeColor = eyeColor
	}

	if msg[29].Type() != MessageElemTypeByte {
		err := fmt.Errorf("rightEyeColor has unexpected type %v", msg[29].Type())
		logger.Println(err)
		return nil, err
	}

	v.RightEyeColor = msg[29].Data().(uint8)

	if v.RightEyeColor == 0 {
		v.RightEyeColor = eyeColor
	}

	if msg[39].Type() != MessageElemTypeInt {
		err := fmt.Errorf("regenCount has unexpected type %v", msg[39].Type())
		logger.Println(err)
		return nil, err
	}

	regenCount := msg[39].Data().(uint32)

	msg = msg[40:]

	if len(msg) < 7*int(regenCount) {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	msg = msg[7*regenCount:]

	if len(msg) < 1 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	if msg[0].Type() != MessageElemTypeInt {
		err := fmt.Errorf("regen2Count has unexpected type %v", msg[0].Type())
		logger.Println(err)
		return nil, err
	}

	regen2Count := msg[0].Data().(uint32)
	msg = msg[1:]

	if len(msg) < 7*int(regen2Count) {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	msg = msg[7*regen2Count:]

	if len(msg) < 10 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	if msg[0].Type() != MessageElemTypeInt {
		err := fmt.Errorf("titleId has unexpected type %v", msg[0].Type())
		logger.Println(err)
		return nil, err
	}

	v.TitleId = msg[0].Data().(uint32)

	if msg[2].Type() != MessageElemTypeInt {
		err := fmt.Errorf("subTitleId has unexpected type %v", msg[2].Type())
		logger.Println(err)
		return nil, err
	}

	v.SubTitleId = msg[2].Data().(uint32)

	if msg[3].Type() != MessageElemTypeInt {
		err := fmt.Errorf("styleTitleId has unexpected type %v", msg[3].Type())
		logger.Println(err)
		return nil, err
	}

	v.StyleTitleId = msg[3].Data().(uint32)

	if msg[4].Type() != MessageElemTypeInt {
		err := fmt.Errorf("styleSubTitleId has unexpected type %v", msg[4].Type())
		logger.Println(err)
		return nil, err
	}

	v.StyleSubTitleId = msg[4].Data().(uint32)

	unk1Count := msg[9].Data().(uint32)
	msg = msg[10:]

	if len(msg) < 2*int(unk1Count) {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	msg = msg[2*unk1Count:]

	if len(msg) < 1 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	if msg[0].Type() != MessageElemTypeInt {
		err := fmt.Errorf("equipItemCount has unexpected type %v", msg[0].Type())
		logger.Println(err)
		return nil, err
	}

	equipItemCount := int(msg[0].Data().(uint32))
	if len(msg) < 2*equipItemCount {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	msg = msg[1:]

	for i := 0; i < equipItemCount; i, msg = i+1, msg[2:] {
		// msg[0] is the Unique ID (Long)
		if msg[0].Type() != MessageElemTypeLong {
			err := fmt.Errorf("equipItemUniqueId has unexpected type %v", msg[0].Type())
			logger.Println(err)
			return nil, err
		}
		uniqueId := msg[0].Data().(uint64)

		if msg[1].Type() != MessageElemTypeBin {
			err := fmt.Errorf("equipItemData has unexpected type %v", msg[1].Type())
			logger.Println(err)
			return nil, err
		}

		b := msg[1].Data().([]byte)
		d, err := EntityItemReader(b)
		if err != nil {
			logger.Println("EntityItemReader failed:", err, i)
			return nil, err
		}

		d.UniqueId = uniqueId
		v.EquipItemMap[d.PocketType] = d

		if msg[2].Type() == MessageElemTypeString {
			// 길드 로브
			msg = msg[1:]
		}
	}

	// 스킬 관련
	if len(msg) < 4 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	if msg[3].Type() != MessageElemTypeInt {
		err := fmt.Errorf("skillCount has unexpected type %v", msg[3].Type())
		logger.Println(err)
		return nil, err
	}

	skillCount := int(msg[3].Data().(uint32))
	msg = msg[4:]

	if len(msg) < skillCount {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	msg = msg[skillCount:]

	// unknown field
	if len(msg) < 2 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	msg = msg[2:]

	// 파티 관련
	if len(msg) < 2 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	msg = msg[2:]

	// pvp 관련
	if len(msg) < 16 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	msg = msg[16:]

	// 컨디션 관련
	if len(msg) < 3 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	if msg[2].Type() != MessageElemTypeInt {
		err := fmt.Errorf("conditionCount has unexpected type %v", msg[2].Type())
		logger.Println(err)
		return nil, err
	}

	conditionCount := int(msg[2].Data().(uint32))
	msg = msg[3:]

	if len(msg) < (conditionCount * 6) {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	for i := 0; i < conditionCount; i, msg = i+1, msg[6:] {
		/*
			uint32 ccId
			uint64 disableAt
			string metadata 나중에 필요할 수 도 있음
			uint64 attackerId
			string unknown1
			string 해제시 메세지?
		*/

		if msg[0].Type() != MessageElemTypeInt {
			err := fmt.Errorf("ccId has unexpected type %v", msg[0].Type())
			logger.Println(err)
			return nil, err
		}

		ccId := msg[0].Data().(uint32)

		if msg[1].Type() != MessageElemTypeLong {
			err := fmt.Errorf("disableAt has unexpected type %v", msg[1].Type())
			logger.Println(err)
			return nil, err
		}

		disableAtRaw := msg[1].Data().(uint64)
		disableAt := util.ParseMabiTime(disableAtRaw).Unix()

		if msg[2].Type() != MessageElemTypeString {
			err := fmt.Errorf("metaData has unexpected type %v", msg[2].Type())
			logger.Println(err)
			return nil, err
		}
		metaData := msg[2].Data().(string)

		if msg[3].Type() != MessageElemTypeLong {
			err := fmt.Errorf("attackerId has unexpected type %v", msg[3].Type())
			logger.Println(err)
			return nil, err
		}

		attackerId := msg[3].Data().(uint64)

		v.CharacterConditionMap[ccId] = &EntityCharacterCondition{
			CCId:       ccId,
			DisableAt:  disableAt,
			MetaData:   metaData,
			AttackerId: attackerId,
		}
	}

	// unknown field
	if len(msg) < 1 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	msg = msg[1:]

	// 길드 관련
	if len(msg) < 19 {
		err := fmt.Errorf("entity appear data is too short %v", curPos())
		logger.Println(err)
		return nil, err
	}

	if msg[1].Type() != MessageElemTypeString {
		err := fmt.Errorf("guildName has unexpected type %v", msg[1].Type())
		logger.Println(err)
		return nil, err
	}

	v.GuildName = msg[1].Data().(string)
	msg = msg[19:]

	// 펫 / 마리오네트 관련
	if IsMarionetteRace(v.RaceId) {
		if len(msg) > 49 && msg[49].Type() == MessageElemTypeLong {
			v.OwnerId = msg[49].Data().(uint64)
		}
		// For marionettes, still advance the pet-related fields if they exist to keep the message slice consistent,
		// but ignore errors if this sub-block is missing/short.
		if len(msg) >= 2 && msg[1].Type() == MessageElemTypeLong {
			msg = msg[2:]
		}
	} else {
		if len(msg) < 2 {
			err := fmt.Errorf("entity appear data is too short %v", curPos())
			logger.Println(err)
			return nil, err
		}

		if msg[1].Type() != MessageElemTypeLong {
			err := fmt.Errorf("ownerId has unexpected type %v", msg[1].Type())
			logger.Println(err)
			return nil, err
		}

		v.OwnerId = msg[1].Data().(uint64)
		msg = msg[2:]
	}

	// --- LOGGING ---
	// logger.Println("========================================")
	// logger.Println("--- Parsed EntityAppear Packet ---")
	// logger.Printf("\tName: %s, Race: %d, Age: %d, Entity ID: %d\n", v.Name, v.RaceId, v.Age, v.Id)
	// logger.Printf("\tLevel: %d / %d (Current/Total)\n", v.CurrentLevel, v.TotalLevel)
	// logger.Printf("\tCombatPower: %f\n", v.CombatPower)
	// logger.Printf("\tHP: %f / %f (%f)\n", v.CurrentHP, v.MaxHP, v.OverflowMaxHP)
	// logger.Printf("\tHP Breakdown (Base/Add): %f / %f\n", v.BaseHP, v.AdditionalHP)
	// logger.Printf("\tVital Surge Shield: %f / %f\n", v.CurrentVitalSurge, v.MaxVitalSurge)
	// logger.Printf("\tEquipment:\n")
	// logger.Println("\tEquiped Items: ", v.EquipItemMap)
	// logger.Println("========================================")
	// --- END LOGGING ---

	// logger.Printf("Successfully Parsed Entity: %v \n Conditions: %v", v.Name, v.CharacterConditionMap)
	return v, nil
}

func ParseEntitiesAppearPacket(p *GamePacket) ([]*EntityInfo, error) {
	entities := []*EntityInfo(nil)
	msg := p.Msg
	if len(msg) < 1 || msg[0].Type() != MessageElemTypeShort {
		return nil, fmt.Errorf("invalid entities appear packet")
	}

	count := int(msg[0].Data().(uint16))
	msg = msg[1:]

	for i := 0; i < count; i++ {
		if len(msg) < 3 {
			break
		}

		if msg[0].Type() != MessageElemTypeShort ||
			msg[1].Type() != MessageElemTypeInt ||
			msg[2].Type() != MessageElemTypeBin {
			logger.Println("invalid packet : expected short, int, bin structure", i)
			continue
		}

		b := msg[2].Data().([]byte)
		// logger.Println("Entity", i, t, msg[1])
		msg = msg[3:]

		_, _, subMsg, err := GamePacketBodyReader(bytes.NewReader(b))
		if err != nil {
			logger.Println("GamePacketBodyReader failed:", err)
			continue
		}

		v, err := ParseEntityAppearPacket(subMsg)
		if err != nil {
			continue
		}

		if v != nil {
			entities = append(entities, v)
		}
	}

	return entities, nil
}

func EntityItemReader(b []byte) (*EntityItem, error) {
	r := new(EntityItem)
	if len(b) < 38 {
		err := fmt.Errorf("item public info data is too short %v", len(b))
		return nil, err
	}

	r.PocketType = le.Uint32(b[0:]) // uint8일듯?
	r.ItemId = le.Uint32(b[4:])
	r.Color1 = le.Uint32(b[8:])
	r.Color2 = le.Uint32(b[12:])
	r.Color3 = le.Uint32(b[16:])
	r.Color4 = le.Uint32(b[20:])
	r.Color5 = le.Uint32(b[24:])
	r.Color6 = le.Uint32(b[28:])
	r.Color7 = le.Uint32(b[32:])
	r.Amount = le.Uint16(b[36:])
	if r.Amount == 0 {
		r.Amount = 1
	}

	return r, nil
}

func getElemTypeName(t MessageElemType) string {
	switch t {
	case MessageElemTypeByte:
		return "Byte"
	case MessageElemTypeShort:
		return "Short"
	case MessageElemTypeInt:
		return "Int"
	case MessageElemTypeLong:
		return "Long"
	case MessageElemTypeFloat:
		return "Float"
	case MessageElemTypeString:
		return "String"
	case MessageElemTypeBin:
		return "Bin"
	default:
		return "Unknown"
	}
}

func IsMarionetteRace(raceId uint32) bool {
	return (raceId >= 990100 && raceId <= 990199) || (raceId >= 990200 && raceId <= 990299)
}
