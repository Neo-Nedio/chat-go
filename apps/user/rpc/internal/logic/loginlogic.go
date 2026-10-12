package logic

import (
	"chat-go/apps/user/rpc/models"
	"chat-go/pkg/ctxdata"
	encrypt "chat-go/pkg/encrypy"
	"chat-go/pkg/xerr"
	"context"
	"time"

	"chat-go/apps/user/rpc/internal/svc"
	"chat-go/apps/user/rpc/user"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *user.LoginReq) (*user.LoginResp, error) {
	u := &models.User{}

	// 1. 查用户
	err := l.svcCtx.CSvc.GetUserByPhone(u, in.Phone)
	switch {
	case err == nil:
		// 查到，继续
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, errors.WithStack(xerr.PhoneNotFound)
	default:
		return nil, errors.Wrapf(xerr.NewDBErr(), "find user by phone err %v, phone %v", err, in.Phone)
	}

	// 2. 密码验证
	if !encrypt.ValidatePasswordHash([]byte(u.Password), []byte(in.Password)) {
		return nil, errors.WithStack(xerr.UserPwdErr)
	}

	// 3. 生成 token
	now := time.Now().Unix()
	token, err := ctxdata.GetJwtToken(
		l.svcCtx.Config.Jwt.AccessSecret,
		now,
		l.svcCtx.Config.Jwt.AccessExpire,
		u.ID,
	)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewServerCommonErr(), "gen jwt token err %v, phone %v", err, in.Phone)
	}

	return &user.LoginResp{
		Id:     u.ID,
		Token:  token,
		Expire: now + l.svcCtx.Config.Jwt.AccessExpire,
	}, nil
}
