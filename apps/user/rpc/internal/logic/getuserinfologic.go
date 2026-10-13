package logic

import (
	"chat-go/apps/user/rpc/models"
	"chat-go/pkg/xerr"
	"context"

	"chat-go/apps/user/rpc/internal/svc"
	"chat-go/apps/user/rpc/user"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserInfoLogic) GetUserInfo(in *user.GetUserInfoReq) (*user.GetUserInfoResp, error) {
	users := make([]models.User, 0, 1)
	ids := []string{in.User}

	err := l.svcCtx.CSvc.GetUserByIds(&users, ids)
	switch {
	case err == nil:
		// 查到，继续
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, errors.WithStack(xerr.IdNotFound)
	default:
		return nil, errors.Wrapf(xerr.NewDBErr(), "find user by id err %v, id %v", err, in.User)
	}

	if len(users) == 0 {
		return nil, errors.WithStack(xerr.IdNotFound)
	}

	ur := users[0]

	var status, sex int32
	if ur.Status != nil {
		status = int32(*ur.Status)
	}
	if ur.Sex != nil {
		sex = int32(*ur.Sex)
	}

	userEntity := user.UserEntity{
		Id:       ur.ID,
		Avatar:   ur.Avatar,
		Nickname: ur.Nickname,
		Phone:    ur.Phone,
		Status:   status,
		Sex:      sex,
	}

	return &user.GetUserInfoResp{User: &userEntity}, nil
}
