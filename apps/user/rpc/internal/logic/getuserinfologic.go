package logic

import (
	"chat-go/apps/user/models"
	"context"

	"chat-go/apps/user/rpc/internal/svc"
	"chat-go/apps/user/rpc/user"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

var ErrUserNotFound = errors.New("这个用户没有")

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

	userEntiy, err := l.svcCtx.UsersModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == models.ErrNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	var resp user.UserEntity
	if err := copier.Copy(&resp, userEntiy); err != nil {
		return nil, err
	}

	return &user.GetUserInfoResp{User: &resp}, nil
}
