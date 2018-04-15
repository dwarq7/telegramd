/*
 *  Copyright (c) 2018, https://github.com/nebulaim
 *  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package chat

import (
	"github.com/nebulaim/telegramd/mtproto"
	"time"
	"github.com/nebulaim/telegramd/biz/base"
	"github.com/nebulaim/telegramd/biz/dal/dataobject"
	"math/rand"
	"github.com/nebulaim/telegramd/biz/dal/dao"
	"fmt"
)

const (
	kChatParticipant = 0
	kChatParticipantCreator = 1
	kChatParticipantAdmin = 2
)

type chatLogicData struct {
	chat         *dataobject.ChatsDO
	participants []dataobject.ChatParticipantsDO
}

//func chatToDO(creatorId int32, chat *mtproto.TLChat) *dataobject.ChatsDO {
//	return &dataobject.ChatsDO{
//		CreatorUserId:    creatorId,
//		RandomId:   rand.Int63(),
//		Title:            chat.GetTitle(),
//		Version:          chat.GetVersion(),
//		ParticipantCount: chat.GetParticipantsCount(),
//	}
//}

//func chatParticipantToDO(chat *mtproto.TLChat, participant *mtproto.ChatParticipant) *dataobject.ChatParticipantsDO {
//	// uId := u.GetInputUser().GetUserId()
//	chatParticipantsDO := &dataobject.ChatParticipantsDO{
//		ChatId: chat.GetId(),
//		UserId: participant.GetData2().GetUserId(),
//		InvitedAt: participant.GetData2().GetInviterId(),
//	}
//
//	return chatParticipantsDO
//
//	// dao.GetChatParticipantsDAO(dao.DB_MASTER).Insert(chatParticipantsDO)
//
//	//chatUserDO.ChatId = chatId
//	//chatUserDO.CreatedAt = base.NowFormatYMDHMS()
//	//chatUserDO.State = 0
//	//chatUserDO.InvitedAt = int32(time.Now().Unix())
//	//chatUserDO.InviterUserId = inviterId
//	//chatUserDO.JoinedAt = chatUserDO.InvitedAt
//	//chatUserDO.UserId = chatUserId
//	//chatUserDO.ParticipantType = participantType
//	//
//	//if participantType == 2 {
//	//	participant2 := mtproto.NewTLChatParticipantCreator()
//	//	participant2.SetUserId(chatUserId)
//	//
//	//	participant = participant2.To_ChatParticipant()
//	//} else if participantType == 1 {
//	//	participant2 := mtproto.NewTLChatParticipantAdmin()
//	//	participant2.SetUserId(chatUserId)
//	//	participant2.SetDate(chatUserDO.InvitedAt)
//	//	participant2.SetInviterId(inviterId)
//	//
//	//	participant = participant2.To_ChatParticipant()
//	//} else if participantType == 0 {
//	//	participant2 := mtproto.NewTLChatParticipant()
//	//	participant2.SetUserId(chatUserId)
//	//	participant2.SetDate(chatUserDO.InvitedAt)
//	//	participant2.SetInviterId(inviterId)
//	//	// participants.Participants = append(participants.Participants, participant.ToChatParticipant())
//	//
//	//	participant = participant2.To_ChatParticipant()
//	//}
//	// return
//}

func makeChatParticipantByDO(do *dataobject.ChatParticipantsDO) (participant *mtproto.ChatParticipant) {
	participant = &mtproto.ChatParticipant{Data2: &mtproto.ChatParticipant_Data{
		UserId:    do.UserId,
		InviterId: do.InviterUserId,
		Date:      do.JoinedAt,
	}}

	switch do.ParticipantType {
	case kChatParticipant:
		participant.Constructor = mtproto.TLConstructor_CRC32_chatParticipant
	case kChatParticipantCreator:
		participant.Constructor = mtproto.TLConstructor_CRC32_chatParticipantCreator
	case kChatParticipantAdmin:
		participant.Constructor = mtproto.TLConstructor_CRC32_chatParticipantAdmin
	default:
		panic("chatParticipant type error.")
	}

	return
}

func NewChatLogicById(chatId int32) (chatData *chatLogicData, err error) {
	chatDO := dao.GetChatsDAO(dao.DB_SLAVE).Select(chatId)
	if chatDO == nil {
		err = mtproto.NewRpcError2(mtproto.TLRpcErrorCodes_CHAT_ID_INVALID)
	} else {
		chatData = &chatLogicData{
			chat: chatDO,
		}
	}
	return
}

func NewChatLogicByCreateChat(creatorId int32, userIds []int32, title string) (chatData *chatLogicData) {
	// TODO(@benqi): 事务
	chatData = &chatLogicData{
		chat: &dataobject.ChatsDO{
			CreatorUserId: creatorId,
			AccessHash:    rand.Int63(),
			// TODO(@benqi): use message_id is randomid
			// RandomId:         base.NextSnowflakeId(),
			ParticipantCount: int32(1 + len(userIds)),
			Title:            title,
			PhotoId:          0,
			Version:          1,
			Date:             int32(time.Now().Unix()),
		},
		participants: make([]dataobject.ChatParticipantsDO, 1+len(userIds)),
	}
	chatData.chat.Id = int32(dao.GetChatsDAO(dao.DB_MASTER).Insert(chatData.chat))

	chatData.participants = make([]dataobject.ChatParticipantsDO, 1 + len(userIds))
	chatData.participants[0].ChatId = chatData.chat.Id
	chatData.participants[0].UserId = creatorId
	chatData.participants[0].ParticipantType = kChatParticipantCreator
	dao.GetChatParticipantsDAO(dao.DB_MASTER).Insert(&chatData.participants[0])

	for i := 0; i < len(userIds); i++ {
		chatData.participants[i+1].ChatId = chatData.chat.Id
		chatData.participants[i+1].UserId = userIds[i]
		chatData.participants[i+1].ParticipantType = kChatParticipant
		chatData.participants[i+1].InviterUserId = creatorId
		chatData.participants[i+1].InvitedAt = chatData.chat.Date
		chatData.participants[i+1].JoinedAt = chatData.chat.Date
		dao.GetChatParticipantsDAO(dao.DB_MASTER).Insert(&chatData.participants[i+1])
	}
	return
}

func (this *chatLogicData) GetChatId() int32 {
	return this.chat.Id
}

func (this *chatLogicData) GetVersion() int32 {
	return this.chat.Version
}

func (this *chatLogicData) MakeCreateChatMessage(creatorId int32) *mtproto.Message {
	idList := this.GetChatParticipantIdList()
	action := &mtproto.TLMessageActionChatCreate{Data2: &mtproto.MessageAction_Data{
		Title: this.chat.Title,
		Users: idList,
	}}

	peer := &base.PeerUtil{
		PeerType: base.PEER_CHAT,
		PeerId:   this.chat.Id,
	}

	message := &mtproto.TLMessageService{Data2: &mtproto.Message_Data{
		Date: this.chat.Date,
		FromId: creatorId,
		ToId: peer.ToPeer(),
		Action: action.To_MessageAction(),
	}}
	return message.To_Message()
}

func (this *chatLogicData) MakeAddUserMessage(inviterId, chatUserId int32) *mtproto.Message {
	// idList := this.GetChatParticipantIdList()
	action := &mtproto.TLMessageActionChatAddUser{Data2: &mtproto.MessageAction_Data{
		Title: this.chat.Title,
		Users: []int32{chatUserId},
	}}

	peer := &base.PeerUtil{
		PeerType: base.PEER_CHAT,
		PeerId:   this.chat.Id,
	}

	message := &mtproto.TLMessageService{Data2: &mtproto.Message_Data{
		Date: this.chat.Date,
		FromId: inviterId,
		ToId: peer.ToPeer(),
		Action: action.To_MessageAction(),
	}}
	return message.To_Message()
}

func (this *chatLogicData) MakeDeleteUserMessage(operatorId, chatUserId int32) *mtproto.Message {
	// idList := this.GetChatParticipantIdList()
	action := &mtproto.TLMessageActionChatDeleteUser{Data2: &mtproto.MessageAction_Data{
		Title:  this.chat.Title,
		UserId: chatUserId,
	}}

	peer := &base.PeerUtil{
		PeerType: base.PEER_CHAT,
		PeerId:   this.chat.Id,
	}

	message := &mtproto.TLMessageService{Data2: &mtproto.Message_Data{
		Date: this.chat.Date,
		FromId: operatorId,
		ToId: peer.ToPeer(),
		Action: action.To_MessageAction(),
	}}
	return message.To_Message()
}

func (this *chatLogicData) MakeChatEditTitleMessage(operatorId int32, title string) *mtproto.Message {
	// idList := this.GetChatParticipantIdList()
	action := &mtproto.TLMessageActionChatEditTitle{Data2: &mtproto.MessageAction_Data{
		Title:  title,
	}}

	peer := &base.PeerUtil{
		PeerType: base.PEER_CHAT,
		PeerId:   this.chat.Id,
	}

	message := &mtproto.TLMessageService{Data2: &mtproto.Message_Data{
		Date: this.chat.Date,
		FromId: operatorId,
		ToId: peer.ToPeer(),
		Action: action.To_MessageAction(),
	}}
	return message.To_Message()
}

func (this *chatLogicData) checkOrLoadChatParticipantList() {
	if len(this.participants) == 0 {
		this.participants = dao.GetChatParticipantsDAO(dao.DB_SLAVE).SelectByChatId(this.chat.Id)
	}
}

func (this *chatLogicData) GetChatParticipantList() []*mtproto.ChatParticipant {
	this.checkOrLoadChatParticipantList()

	participantList := make([]*mtproto.ChatParticipant, 0, len(this.participants))
	for i := 0; i < len(this.participants); i++ {
		if this.participants[i].State == 0 {
			participantList = append(participantList, makeChatParticipantByDO(&this.participants[i]))
		}
	}
	return participantList
}


func (this *chatLogicData) GetChatParticipantIdList() []int32 {
	this.checkOrLoadChatParticipantList()

	idList := make([]int32, 0, len(this.participants))
	for i := 0; i < len(this.participants); i++  {
		if this.participants[i].State == 0 {
			idList = append(idList, this.participants[i].UserId)
		}
	}
	return idList
}

func (this *chatLogicData) GetChatParticipants() *mtproto.TLChatParticipants {
	this.checkOrLoadChatParticipantList()

	return &mtproto.TLChatParticipants{Data2: &mtproto.ChatParticipants_Data{
		ChatId: this.chat.Id,
		Participants: this.GetChatParticipantList(),
		Version: this.chat.Version,
	}}
}

func (this *chatLogicData) AddChatUser(inviterId, userId int32) error {
	this.checkOrLoadChatParticipantList()

	// TODO(@benqi): check userId exisits
	var founded = -1
	for i := 0; i < len(this.participants); i++ {
		if userId == this.participants[i].UserId {
			if this.participants[i].State == 1 {
				founded = i
			} else {
				return fmt.Errorf("userId exisits")
			}
		}
	}

	var now = int32(time.Now().Unix())

	if founded != -1 {
		this.participants[founded].State = 0
		dao.GetChatParticipantsDAO(dao.DB_MASTER).Update(inviterId, now, now, this.participants[founded].Id)
	} else {
		chatParticipant := &dataobject.ChatParticipantsDO{
			ChatId:          this.chat.Id,
			UserId:          userId,
			ParticipantType: kChatParticipant,
			InviterUserId:   inviterId,
			InvitedAt:       now,
			JoinedAt:        now,
		}
		chatParticipant.Id = int32(dao.GetChatParticipantsDAO(dao.DB_MASTER).Insert(chatParticipant))
		this.participants = append(this.participants, *chatParticipant)
	}

	// update chat
	this.chat.ParticipantCount += 1
	this.chat.Version += 1
	this.chat.Date = now
	dao.GetChatsDAO(dao.DB_MASTER).UpdateParticipantCount(this.chat.ParticipantCount, now, this.chat.Id)

	return nil
}

func (this *chatLogicData) findChatParticipant(selfUserId int32) (int, *dataobject.ChatParticipantsDO) {
	for i := 0; i < len(this.participants); i++ {
		if this.participants[i].UserId == selfUserId {
			return i, &this.participants[i]
		}
	}
	return -1, nil
}

func (this *chatLogicData) ToChat(selfUserId int32) *mtproto.Chat {
	// TODO(@benqi): kicked:flags.1?true left:flags.2?true admins_enabled:flags.3?true admin:flags.4?true deactivated:flags.5?true

	var forbidden = false
	for i := 0; i < len(this.participants); i++ {
		if this.participants[i].UserId == selfUserId && this.participants[i].State == 1 {
			forbidden = true
			break
		}
	}

	if forbidden {
		chat := &mtproto.TLChatForbidden{Data2: &mtproto.Chat_Data{
			Id:    this.chat.Id,
			Title: this.chat.Title,
		}}
		return chat.To_Chat()
	} else {
		chat := &mtproto.TLChat{Data2: &mtproto.Chat_Data{
			Creator:           this.chat.CreatorUserId == selfUserId,
			Id:                this.chat.Id,
			Title:             this.chat.Title,
			Photo:             mtproto.NewTLChatPhotoEmpty().To_ChatPhoto(),
			ParticipantsCount: this.chat.ParticipantCount,
			Date:              this.chat.Date,
			Version:           this.chat.Version,
		}}
		return chat.To_Chat()
	}
}

func (this *chatLogicData) DeleteChatUser(userId int32) error {
	this.checkOrLoadChatParticipantList()

	var found = -1
	for i := 0; i < len(this.participants); i++ {
		if userId == this.participants[i].UserId {
			if this.participants[i].State == 0 {
				found = i
			}
			break
		}
	}

	if found == -1 {
		return fmt.Errorf("not found delete chat user")
	}

	this.participants[found].State = 1
	this.participants = append(this.participants[:found], this.participants[found+1:]...)
	dao.GetChatParticipantsDAO(dao.DB_MASTER).DeleteChatUser(this.participants[found].Id)

	var now = int32(time.Now().Unix())
	this.chat.ParticipantCount -= 1
	this.chat.Version += 1
	this.chat.Date = now
	dao.GetChatsDAO(dao.DB_MASTER).UpdateParticipantCount(this.chat.ParticipantCount, now, this.chat.Id)

	return nil
}

func (this *chatLogicData) EditChatTitle(editUserId int32, title string) error {
	this.checkOrLoadChatParticipantList()

	_, participant := this.findChatParticipant(editUserId)

	if participant == nil {
		return mtproto.NewRpcError2(mtproto.TLRpcErrorCodes_PARTICIPANT_NOT_EXISTS)
	}

	// check editUserId is creator or admin
	if participant.ParticipantType == kChatParticipant {
		return mtproto.NewRpcError2(mtproto.TLRpcErrorCodes_NO_EDIT_CHAT_PERMISSION)
	}

	if this.chat.Title == title {
		return mtproto.NewRpcError2(mtproto.TLRpcErrorCodes_CHAT_NOT_MODIFIED)
	}

	this.chat.Title = title
	this.chat.Date = int32(time.Now().Unix())
	this.chat.Version += 1

	dao.GetChatsDAO(dao.DB_MASTER).UpdateTitle(title, this.chat.Date, this.chat.Id)
	return nil
}

func (this *chatLogicData) EditChatPhoto(editUserId int32, photoId int64) error {
	this.checkOrLoadChatParticipantList()

	_, participant := this.findChatParticipant(editUserId)

	if participant == nil {
		return mtproto.NewRpcError2(mtproto.TLRpcErrorCodes_PARTICIPANT_NOT_EXISTS)
	}

	// check editUserId is creator or admin
	if participant.ParticipantType == kChatParticipant {
		return mtproto.NewRpcError2(mtproto.TLRpcErrorCodes_NO_EDIT_CHAT_PERMISSION)
	}

	if this.chat.PhotoId == photoId {
		return mtproto.NewRpcError2(mtproto.TLRpcErrorCodes_CHAT_NOT_MODIFIED)
	}

	this.chat.PhotoId = photoId
	this.chat.Date = int32(time.Now().Unix())
	this.chat.Version += 1

	dao.GetChatsDAO(dao.DB_MASTER).UpdatePhotoId(photoId, this.chat.Date, this.chat.Id)
	return nil
}

func (this *chatLogicData) EditChatAdmin(editUserId int32) error {
	this.checkOrLoadChatParticipantList()

	_, participant := this.findChatParticipant(editUserId)

	if participant == nil {
		return mtproto.NewRpcError2(mtproto.TLRpcErrorCodes_PARTICIPANT_NOT_EXISTS)
	}

	// check editUserId is creator or admin
	if participant.ParticipantType == kChatParticipant {
		return mtproto.NewRpcError2(mtproto.TLRpcErrorCodes_NO_EDIT_CHAT_PERMISSION)
	}

	// ...

	return nil
}