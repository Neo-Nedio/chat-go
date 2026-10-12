package status

// HandlerResult 请求处理结果（好友申请、入群申请等的状态）
type HandlerResult int8

const (
	PendingHandlerResult HandlerResult = 1 + iota //待处理
	PassHandlerResult                             //已通过
	RefuseHandlerResult                           //	已拒绝
	CancelHandlerResult                           //已取消
)

// GroupMemberRoleLevel 群成员角色等级
type GroupMemberRoleLevel int8

const (
	GroupMemberRoleLevelOwner GroupMemberRoleLevel = 1 + iota
	GroupMemberRoleLevelAdmin
	GroupMemberRoleLevelMember
)

// GroupType 群类型
type GroupType int8

const (
	GroupTypeNormal GroupType = 1 + iota
	GroupTypePrivate
	GroupTypePublic
)
