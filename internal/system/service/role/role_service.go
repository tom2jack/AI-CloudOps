package role

/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao/menu"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao/role"
	"go.uber.org/zap"
)

type RoleService interface {
	GetAllRoleList(ctx context.Context) ([]*model.Role, error)
	CreateRole(ctx context.Context, roles model.Role) error
	UpdateRole(ctx context.Context, roles model.Role) error
	SetRoleStatus(ctx context.Context, id int, status string) error
	DeleteRole(ctx context.Context, id string) error
}

type roleService struct {
	menuDao menu.MenuDAO
	roleDao role.RoleDAO
	l       *zap.Logger
}

func NewRoleService(menuDao menu.MenuDAO, roleDao role.RoleDAO, l *zap.Logger) RoleService {
	return &roleService{
		menuDao: menuDao,
		roleDao: roleDao,
		l:       l,
	}
}

// GetAllRoleList 获取所有角色列表
func (r *roleService) GetAllRoleList(ctx context.Context) ([]*model.Role, error) {
	return r.roleDao.GetAllRoles(ctx)
}

// CreateRole 创建新角色
func (r *roleService) CreateRole(ctx context.Context, role model.Role) error {
	// 通过菜单ID列表获取菜单对象
	menus, err := r.getMenusByIDs(ctx, role.MenuIds)
	if err != nil {
		return err
	}

	// 将菜单分配给角色
	role.Menus = menus

	return r.roleDao.CreateRole(ctx, &role)
}

// UpdateRole 更新角色信息
func (r *roleService) UpdateRole(ctx context.Context, role model.Role) error {
	// 通过菜单ID列表获取菜单对象
	menus, err := r.getMenusByIDs(ctx, role.MenuIds)
	if err != nil {
		return err
	}

	// 更新角色菜单
	role.Menus = menus

	return r.roleDao.UpdateRole(ctx, &role)
}

func (r *roleService) SetRoleStatus(ctx context.Context, roleID int, status string) error {
	return r.roleDao.UpdateRoleStatus(ctx, roleID, status)
}

func (r *roleService) DeleteRole(ctx context.Context, id string) error {
	return r.roleDao.DeleteRole(ctx, id)
}

// getMenusByIDs 根据菜单ID列表获取对应的菜单对象
func (r *roleService) getMenusByIDs(ctx context.Context, menuIds []int) ([]*model.Menu, error) {
	menus := make([]*model.Menu, 0)

	for _, menuId := range menuIds {
		// 根据ID获取菜单信息
		menu, err := r.menuDao.GetMenuByID(ctx, int(menuId))
		if err != nil {
			r.l.Error("GetMenuByID failed", zap.Error(err))
			return nil, err
		}

		menus = append(menus, menu)
	}

	return menus, nil
}
