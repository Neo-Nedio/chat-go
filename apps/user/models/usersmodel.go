package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UsersModel = (*customUsersModel)(nil)

type (
	// UsersModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUsersModel.
	UsersModel interface {
		usersModel
		FindByPhone(ctx context.Context, phone string) (*Users, error)
		ListByName(ctx context.Context, name string) ([]*Users, error)
		ListByIds(ctx context.Context, ids []string) ([]*Users, error)
	}

	customUsersModel struct {
		*defaultUsersModel
	}
)

// NewUsersModel returns a model for the database table.
func NewUsersModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UsersModel {
	return &customUsersModel{
		defaultUsersModel: newUsersModel(conn, c, opts...),
	}
}

// FindByPhone finds one user by phone number.
func (m *customUsersModel) FindByPhone(ctx context.Context, phone string) (*Users, error) {
	var resp Users
	query := fmt.Sprintf("select %s from %s where `phone` = ? limit 1", usersRows, m.table)
	if err := m.QueryRowNoCacheCtx(ctx, &resp, query, phone); err != nil {
		if err == sqlc.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &resp, nil
}

// ListByName searches users by nickname.
func (m *customUsersModel) ListByName(ctx context.Context, name string) ([]*Users, error) {
	query := fmt.Sprintf("select %s from %s where `nickname` like ?", usersRows, m.table)
	resp := make([]*Users, 0)
	if err := m.QueryRowsNoCacheCtx(ctx, &resp, query, "%"+name+"%"); err != nil {
		return nil, err
	}

	return resp, nil
}

// ListByIds finds users whose ids are in the supplied list.
func (m *customUsersModel) ListByIds(ctx context.Context, ids []string) ([]*Users, error) {
	if len(ids) == 0 {
		return []*Users{}, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	query := fmt.Sprintf("select %s from %s where `id` in (%s)", usersRows, m.table, placeholders)
	resp := make([]*Users, 0, len(ids))
	if err := m.QueryRowsNoCacheCtx(ctx, &resp, query, args...); err != nil {
		return nil, err
	}

	return resp, nil
}
