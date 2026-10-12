package logic

import (
	"chat-go/apps/user/rpc/models"
	"chat-go/pkg/ctxdata"
	encrypt "chat-go/pkg/encrypy"
	"chat-go/pkg/suid"
	"chat-go/pkg/utils"
	"chat-go/pkg/xerr"
	"context"
	"time"

	"chat-go/apps/user/rpc/internal/svc"
	"chat-go/apps/user/rpc/user"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *user.RegisterReq) (*user.RegisterResp, error) {
	u := models.User{}

	// 1. 检查用户是否存在
	err := l.svcCtx.CSvc.GetUserByPhone(&u, in.Phone)
	switch {
	case err == nil:
		return nil, xerr.New(xerr.ServerCommonError, "phone already exists")
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 查不到，继续
	default:
		return nil, errors.Wrapf(xerr.NewDBErr(), "find user by phone err %v, phone %v", err, in.Phone)
	}

	// 2. 构造新用户
	newUser := &models.User{
		ID:       suid.GenerateID(),
		Avatar:   in.Avatar,
		Nickname: in.Nickname,
		Phone:    in.Phone,
		Status:   utils.ConvertToInt8(0),
		Sex:      utils.ConvertToInt8(in.Sex),
	}

	if in.Password != "" {
		pass, hashErr := encrypt.GenPasswordHash([]byte(in.Password))
		if hashErr != nil {
			return nil, errors.Wrapf(xerr.NewServerCommonErr(), "passwordHash gen err %v", hashErr)
		}
		newUser.Password = string(pass)
	}

	// 3. 保存
	if err := l.svcCtx.CSvc.CreateUser(newUser); err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "save user failed, req %v, err %v", in, err)
	}

	// 4. 生成 token
	now := time.Now().Unix()
	token, err := ctxdata.GetJwtToken(
		l.svcCtx.Config.Jwt.AccessSecret,
		now,
		l.svcCtx.Config.Jwt.AccessExpire,
		newUser.ID,
	)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewServerCommonErr(), "gen jwt token err %v, phone %v", err, in.Phone)
	}

	return &user.RegisterResp{
		Token:  token,
		Expire: now + l.svcCtx.Config.Jwt.AccessExpire,
	}, nil
}
