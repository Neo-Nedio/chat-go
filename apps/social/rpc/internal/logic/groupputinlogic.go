package logic

import (
	"chat-go/apps/social/rpc/internal/svc"
	"chat-go/apps/social/rpc/social"
	"chat-go/apps/social/socialmodels"
	"chat-go/pkg/constants"
	"chat-go/pkg/xerr"
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GroupPutinLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGroupPutinLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GroupPutinLogic {
	return &GroupPutinLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GroupPutin 处理入群申请/邀请
// 三种场景：
//  1. 普通用户申请入群
//  2. 群成员邀请他人入群
//  3. 群管理员/群主邀请他人入群
func (l *GroupPutinLogic) GroupPutin(in *social.GroupPutinReq) (*social.GroupPutinResp, error) {

	var (
		inviteGroupMember *socialmodels.GroupMembers // 邀请人的群成员记录
		userGroupMember   *socialmodels.GroupMembers // 申请人的群成员记录
		groupInfo         *socialmodels.Groups       // 群信息

		err error
	)

	// ========== 1. 检查申请人是否已经是群成员 ==========
	// 查申请人（in.ReqId）在群里（in.GroupId）是否已有记录
	userGroupMember, err = l.svcCtx.GroupMembersModel.FindByGroudIdAndUserId(l.ctx, in.ReqId, in.GroupId)
	// ErrNotFound 是正常情况（不是成员），其它错误才是真错误
	if err != nil && !errors.Is(err, socialmodels.ErrNotFound) {
		return nil, errors.Wrapf(xerr.NewDBErr(), "find group member by groud id and  req id err %v, req %v, %v", err,
			in.GroupId, in.ReqId)
	}
	// 已经是成员 → 直接返回，不用再申请
	if userGroupMember != nil {
		return &social.GroupPutinResp{}, nil
	}

	// ========== 2. 检查是否已有待处理的申请 ==========
	// 查申请人在这个群里是否已有申请记录
	groupReq, err := l.svcCtx.GroupRequestsModel.FindByGroupIdAndReqId(l.ctx, in.GroupId, in.ReqId)
	if err != nil && !errors.Is(err, socialmodels.ErrNotFound) {
		return nil, errors.Wrapf(xerr.NewDBErr(), "find group req by groud id and user id err %v, req %v, %v", err,
			in.GroupId, in.ReqId)
	}
	// 已有申请 → 直接返回，防重复申请
	if groupReq != nil {
		return &social.GroupPutinResp{}, nil
	}

	// ========== 3. 构造申请记录 ==========
	// 用 sql.NullXxx 包装，因为 proto 字段可能是零值，
	// 用 Valid: true 明确表示"有值"
	groupReq = &socialmodels.GroupRequests{
		ReqId:   in.ReqId,   // 申请人
		GroupId: in.GroupId, // 群 ID
		ReqMsg: sql.NullString{
			String: in.ReqMsg, // 申请留言
			Valid:  true,
		},
		ReqTime: sql.NullTime{
			Time:  time.Unix(in.ReqTime, 0), // 申请时间
			Valid: true,
		},
		JoinSource: sql.NullInt64{
			Int64: int64(in.JoinSource), // 入群方式（申请/邀请）
			Valid: true,
		},
		InviterUserId: sql.NullString{
			String: in.InviterUid, // 邀请人
			Valid:  true,
		},
		HandleResult: sql.NullInt64{
			Int64: int64(constants.NoHandlerResult), // 默认待处理
			Valid: true,
		},
	}

	// ========== 4. 定义闭包：建群成员 ==========
	// 只有前面没出错时才执行
	createGroupMember := func() {
		if err != nil {
			return
		}
		err = l.createGroupMember(in)
	}

	// ========== 5. 查群信息 ==========
	groupInfo, err = l.svcCtx.GroupsModel.FindOne(l.ctx, in.GroupId)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "find group by groud id err %v, req %v", err, in.GroupId)
	}

	// ========== 6. 群不需要验证 → 直接通过 ==========
	if !groupInfo.IsVerify {
		// defer：函数返回前建成员（此时 createGroupReq 已执行完）
		defer createGroupMember()

		// 申请状态设为"已通过"
		groupReq.HandleResult = sql.NullInt64{
			Int64: int64(constants.PassHandlerResult),
			Valid: true,
		}

		// 插入申请记录，isPass=true 返回带 GroupId 的响应
		return l.createGroupReq(groupReq, true)
	}

	// ========== 7. 群需要验证，判断入群方式 ==========
	// 7.1 用户主动申请 → 待审核
	if constants.GroupJoinSource(in.JoinSource) == constants.PutInGroupJoinSource {
		return l.createGroupReq(groupReq, false) // isPass=false，状态保持待处理
	}

	// 7.2 邀请方式 → 查邀请人的群成员记录
	inviteGroupMember, err = l.svcCtx.GroupMembersModel.FindByGroudIdAndUserId(l.ctx, in.InviterUid, in.GroupId)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "find group member by groud id and user id err %v, req %v",
			in.InviterUid, in.GroupId)
	}

	// 7.3 邀请人是群主或管理员 → 直接通过
	if constants.GroupRoleLevel(inviteGroupMember.RoleLevel) == constants.CreatorGroupRoleLevel ||
		constants.GroupRoleLevel(inviteGroupMember.RoleLevel) == constants.ManagerGroupRoleLevel {
		// defer：返回前建成员
		defer createGroupMember()

		// 状态设为已通过
		groupReq.HandleResult = sql.NullInt64{
			Int64: int64(constants.PassHandlerResult),
			Valid: true,
		}
		// 记录处理人（这里是邀请人）
		groupReq.HandleUserId = sql.NullString{
			String: in.InviterUid,
			Valid:  true,
		}
		return l.createGroupReq(groupReq, true)
	}

	// 7.4 邀请人是普通成员 → 需要审核
	return l.createGroupReq(groupReq, false)
}

// createGroupReq 插入群申请记录
// isPass=true  → 返回带 GroupId 的响应（表示已通过）
// isPass=false → 返回空响应（表示待审核）
func (l *GroupPutinLogic) createGroupReq(groupReq *socialmodels.GroupRequests, isPass bool) (*social.GroupPutinResp, error) {

	_, err := l.svcCtx.GroupRequestsModel.Insert(l.ctx, groupReq)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewDBErr(), "insert group req err %v req %v", err, groupReq)
	}

	// 群申请通过后返回群ID
	if isPass {
		return &social.GroupPutinResp{GroupId: groupReq.GroupId}, nil
	}

	return &social.GroupPutinResp{}, nil
}

// createGroupMember 把申请人加进群
func (l *GroupPutinLogic) createGroupMember(in *social.GroupPutinReq) error {
	groupMember := &socialmodels.GroupMembers{
		GroupId:     in.GroupId,                           // 群 ID
		UserId:      in.ReqId,                             // 申请人变成群成员
		RoleLevel:   int(constants.AtLargeGroupRoleLevel), // 默认普通成员
		OperatorUid: in.InviterUid,                        // 操作人
	}
	_, err := l.svcCtx.GroupMembersModel.Insert(l.ctx, nil, groupMember)
	if err != nil {
		return errors.Wrapf(xerr.NewDBErr(), "insert friend err %v req %v", err, groupMember)
	}

	return nil
}
