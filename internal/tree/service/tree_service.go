package service

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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/ecs"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/elb"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/rds"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/tree_node"
	"github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

var (
	ErrResourceBound    = errors.New("ECS 资源被绑定，无法删除")
	ErrResourceNotFound = errors.New("ECS 资源未找到")
)

type TreeService interface {
	ListTreeNodes(ctx context.Context) ([]*model.TreeNode, error)
	SelectTreeNode(ctx context.Context, level int, levelLt int) ([]*model.TreeNode, error)
	GetTopTreeNode(ctx context.Context) ([]*model.TreeNode, error)
	ListLeafTreeNodes(ctx context.Context) ([]*model.TreeNode, error)
	CreateTreeNode(ctx context.Context, obj *model.TreeNode) error
	DeleteTreeNode(ctx context.Context, id int) error
	UpdateTreeNode(ctx context.Context, obj *model.TreeNode) error
	GetChildrenTreeNodes(ctx context.Context, pid int) ([]*model.TreeNode, error)

	GetEcsUnbindList(ctx context.Context) ([]*model.ResourceEcs, error)
	GetEcsList(ctx context.Context) ([]*model.ResourceEcs, error)
	GetElbUnbindList(ctx context.Context) ([]*model.ResourceElb, error)
	GetElbList(ctx context.Context) ([]*model.ResourceElb, error)
	GetRdsUnbindList(ctx context.Context) ([]*model.ResourceRds, error)
	GetRdsList(ctx context.Context) ([]*model.ResourceRds, error)
	GetAllResourcesByType(ctx context.Context, nid int, resourceType string, page int, size int) ([]*model.ResourceTree, error)
	//GetAllResources(ctx context.Context) ([]*model.ResourceTree, error)

	BindEcs(ctx context.Context, ecsID int, treeNodeID int) error
	BindElb(ctx context.Context, elbID int, treeNodeID int) error
	BindRds(ctx context.Context, rdsID int, treeNodeID int) error
	UnBindEcs(ctx context.Context, ecsID int, treeNodeID int) error
	UnBindElb(ctx context.Context, elbID int, treeNodeID int) error
	UnBindRds(ctx context.Context, rdsID int, treeNodeID int) error

	CreateEcsResource(ctx context.Context, obj *model.ResourceEcs) error
	UpdateEcsResource(ctx context.Context, obj *model.ResourceEcs) error
	DeleteEcsResource(ctx context.Context, id int) error
}

type treeService struct {
	ecsDao  ecs.TreeEcsDAO
	elbDao  elb.TreeElbDAO
	rdsDao  rds.TreeRdsDAO
	nodeDao tree_node.TreeNodeDAO
	userDao dao.UserDAO
	l       *zap.Logger
}

// NewTreeService 构造函数
func NewTreeService(ecsDao ecs.TreeEcsDAO, elbDao elb.TreeElbDAO, rdsDao rds.TreeRdsDAO, nodeDao tree_node.TreeNodeDAO, l *zap.Logger, userDao dao.UserDAO) TreeService {
	return &treeService{
		ecsDao:  ecsDao,
		elbDao:  elbDao,
		rdsDao:  rdsDao,
		nodeDao: nodeDao,
		userDao: userDao,
		l:       l,
	}
}

// ListTreeNodes 获取所有树节点并构建树结构
func (ts *treeService) ListTreeNodes(ctx context.Context) ([]*model.TreeNode, error) {
	// 获取所有节点
	nodes, err := ts.nodeDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("ListTreeNodes 获取所有节点失败", zap.Error(err))
		return nil, err
	}

	// 初始化顶层节点切片和父节点到子节点的映射
	var topNodes []*model.TreeNode
	childrenMap := make(map[int][]*model.TreeNode)

	// 遍历所有节点，设置 Key 属性并分类为顶层节点或子节点
	for _, node := range nodes {
		// 设置节点的 Key 属性
		node.Key = strconv.Itoa(node.ID)

		if node.Pid == 0 {
			// 如果父节点 ID 为 0，则为顶层节点，添加到 topNodes 切片中
			topNodes = append(topNodes, node)
		} else {
			// 否则，将节点添加到对应父节点的子节点列表中
			childrenMap[node.Pid] = append(childrenMap[node.Pid], node)
		}
	}

	// 遍历所有节点，将子节点关联到各自的父节点
	for _, node := range nodes {
		if children, exists := childrenMap[node.ID]; exists {
			node.Children = children
		}
	}

	return topNodes, nil
}

func (ts *treeService) SelectTreeNode(ctx context.Context, level int, levelLt int) ([]*model.TreeNode, error) {
	// 从数据库中获取所有树节点
	nodes, err := ts.nodeDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("SelectTreeNode failed to retrieve nodes", zap.Error(err))
		return nil, err
	}

	filteredNodes := make([]*model.TreeNode, 0, len(nodes))

	for _, node := range nodes {
		// 如果指定了具体层级，且节点的层级不匹配，则跳过该节点
		if level > 0 && node.Level != level {
			continue
		}

		// 如果指定了最大层级，且节点的层级超过该值，则跳过该节点
		if levelLt > 0 && node.Level > levelLt {
			continue
		}

		// 设置节点的 Value 属性为其 ID
		node.Value = node.ID

		filteredNodes = append(filteredNodes, node)
	}

	return filteredNodes, nil
}

func (ts *treeService) GetTopTreeNode(ctx context.Context) ([]*model.TreeNode, error) {
	// level = 1, 顶级节点
	nodes, err := ts.nodeDao.GetByLevel(ctx, 1)
	if err != nil {
		ts.l.Error("GetTopTreeNode failed", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (ts *treeService) ListLeafTreeNodes(ctx context.Context) ([]*model.TreeNode, error) {
	// 从数据库中获取所有树节点
	nodes, err := ts.nodeDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("ListLeafTreeNodes 获取所有树节点失败", zap.Error(err))
		return nil, err
	}

	leafNodes := make([]*model.TreeNode, 0, len(nodes))

	// 遍历所有节点，筛选出叶子节点
	for _, node := range nodes {
		// 如果 BindEcs 为空或长度为 0，则不是叶子节点，跳过
		if len(node.BindEcs) == 0 {
			continue
		}
		leafNodes = append(leafNodes, node)
	}

	return leafNodes, nil
}

func (ts *treeService) CreateTreeNode(ctx context.Context, obj *model.TreeNode) error {
	// 如果是顶级节点，并且层级为 1
	if obj.Pid == 0 && obj.Level == 1 {
		return ts.nodeDao.Create(ctx, obj)
	}

	// 获取父节点
	node, err := ts.nodeDao.GetByID(ctx, obj.Pid)
	if err != nil {
		ts.l.Error("获取父节点失败", zap.Error(err))
		return errors.New("获取父节点失败")
	}

	// 父节点不存在
	if node == nil {
		errMsg := "父节点不存在"
		ts.l.Error(errMsg)
		return errors.New(errMsg)
	}

	// 检查层级是否超过父节点的允许范围
	if obj.Level >= node.Level+2 {
		errMsg := "创建节点失败，节点层级超出限制"
		ts.l.Error(errMsg)
		return errors.New(errMsg)
	}

	// 检查父节点是否为叶子节点
	if node.IsLeaf == 1 {
		errMsg := "创建节点失败，父节点为叶子节点，无法添加子节点"
		ts.l.Error(errMsg)
		return errors.New(errMsg)
	}

	// 创建新节点
	return ts.nodeDao.Create(ctx, obj)
}

func (ts *treeService) DeleteTreeNode(ctx context.Context, id int) error {
	// 获取节点信息
	treeNode, err := ts.nodeDao.GetByID(ctx, id)
	if err != nil {
		ts.l.Error("DeleteTreeNode failed: 获取节点失败", zap.Error(err))
		return err
	}

	// 检查节点是否存在
	if treeNode == nil {
		ts.l.Error("DeleteTreeNode failed: 节点不存在", zap.Error(errors.New("节点不存在")))
		return errors.New("节点不存在")
	}

	// 检查是否有子节点
	hasChildren, err := ts.nodeDao.HasChildren(ctx, id)
	if err != nil {
		ts.l.Error("DeleteTreeNode failed: 检查子节点失败", zap.Error(err))
		return err
	}
	if hasChildren {
		ts.l.Error("DeleteTreeNode failed: 节点有子节点", zap.Error(errors.New(constants.ErrorTreeNodeHasChildren)))
		return errors.New(constants.ErrorTreeNodeHasChildren)
	}

	// 删除节点
	if err := ts.nodeDao.Delete(ctx, id); err != nil {
		ts.l.Error("DeleteTreeNode failed: 删除节点失败", zap.Error(err))
		return err
	}

	return nil
}

func (ts *treeService) UpdateTreeNode(ctx context.Context, obj *model.TreeNode) error {
	var (
		usersOpsAdmin []*model.User
		usersRdAdmin  []*model.User
		usersRdMember []*model.User
		err           error
	)

	// 获取运维负责人
	usersOpsAdmin, err = ts.fetchUsers(ctx, obj.OpsAdminUsers, "OpsAdmin")
	if err != nil {
		ts.l.Error("UpdateTreeNode 获取 OpsAdmin 用户信息失败", zap.Error(err))
		return err
	}

	// 获取研发负责人
	usersRdAdmin, err = ts.fetchUsers(ctx, obj.RdAdminUsers, "RdAdmin")
	if err != nil {
		ts.l.Error("UpdateTreeNode 获取 RdAdmin 用户信息失败", zap.Error(err))
		return err
	}

	// 获取研发工程师
	usersRdMember, err = ts.fetchUsers(ctx, obj.RdMemberUsers, "RdMember")
	if err != nil {
		ts.l.Error("UpdateTreeNode 获取 RdMember 用户信息失败", zap.Error(err))
		return err
	}

	// 更新树节点的用户信息
	obj.OpsAdmins = usersOpsAdmin
	obj.RdAdmins = usersRdAdmin
	obj.RdMembers = usersRdMember

	// 持久化树节点更新
	if err := ts.nodeDao.Update(ctx, obj); err != nil {
		ts.l.Error("UpdateTreeNode 持久化树节点信息失败", zap.Error(err))
		return errors.New("持久化树节点信息失败")
	}

	return nil
}

func (ts *treeService) GetChildrenTreeNodes(ctx context.Context, pid int) ([]*model.TreeNode, error) {
	childrenNodes, err := ts.nodeDao.GetByPid(ctx, pid)
	if err != nil {
		ts.l.Error("GetChildrenTreeNodes failed", zap.Error(err))
		return nil, err
	}

	return childrenNodes, nil
}

func (ts *treeService) GetEcsUnbindList(ctx context.Context) ([]*model.ResourceEcs, error) {
	ecs, err := ts.ecsDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("GetEcsUnbindList failed", zap.Error(err))
		return nil, err
	}

	// 筛选出未绑定的 ECS 资源
	unbindEcs := make([]*model.ResourceEcs, 0, len(ecs))
	for _, e := range ecs {
		if len(e.BindNodes) == 0 {
			unbindEcs = append(unbindEcs, e)
		}
	}

	return unbindEcs, nil
}

func (ts *treeService) GetEcsList(ctx context.Context) ([]*model.ResourceEcs, error) {
	list, err := ts.ecsDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("GetEcsList failed", zap.Error(err))
		return nil, err
	}

	return list, nil
}

func (ts *treeService) GetElbUnbindList(ctx context.Context) ([]*model.ResourceElb, error) {
	elb, err := ts.elbDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("GetElbUnbindList failed", zap.Error(err))
		return nil, err
	}

	// 筛选出未绑定的 ELB 资源
	unbindElb := make([]*model.ResourceElb, 0, len(elb))
	for _, e := range elb {
		if len(e.BindNodes) == 0 {
			unbindElb = append(unbindElb, e)
		}
	}

	return unbindElb, nil
}

func (ts *treeService) GetElbList(ctx context.Context) ([]*model.ResourceElb, error) {
	list, err := ts.elbDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("GetElbList failed", zap.Error(err))
		return nil, err
	}

	return list, nil
}

func (ts *treeService) GetRdsUnbindList(ctx context.Context) ([]*model.ResourceRds, error) {
	rds, err := ts.rdsDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("GetRdsUnbindList failed", zap.Error(err))
		return nil, err
	}

	// 筛选出未绑定的 RDS 资源
	unbindRds := make([]*model.ResourceRds, 0, len(rds))
	for _, e := range rds {
		if len(e.BindNodes) == 0 {
			unbindRds = append(unbindRds, e)
		}
	}

	return unbindRds, nil
}

func (ts *treeService) GetRdsList(ctx context.Context) ([]*model.ResourceRds, error) {
	list, err := ts.rdsDao.GetAll(ctx)
	if err != nil {
		ts.l.Error("GetRdsList failed", zap.Error(err))
		return nil, err
	}

	return list, nil
}

func (ts *treeService) getChildrenTreeNodeIds(ctx context.Context, nid int) map[int]struct{} {
	nodes, err := ts.nodeDao.GetAllNoPreload(ctx)
	if err != nil {
		ts.l.Error("GetChildrenTreeNodeIds failed", zap.Error(err))
		return nil
	}

	nodeMap := make(map[int]*model.TreeNode)
	childrenMap := make(map[int][]*model.TreeNode)
	for _, node := range nodes {
		nodeMap[node.ID] = node
		childrenMap[node.Pid] = append(childrenMap[node.Pid], node)
	}

	idMp := make(map[int]struct{})

	var dfs func(nodeId int)
	dfs = func(nodeId int) {
		node, exists := nodeMap[nodeId]
		if !exists {
			return
		}

		if node.IsLeaf == 1 {
			idMp[node.ID] = struct{}{}
			return
		}

		for _, child := range childrenMap[nodeId] {
			dfs(child.ID)
		}
	}

	dfs(nid)

	return idMp
}

func (ts *treeService) GetAllResourcesByType(ctx context.Context, nid int, resourceType string, page, size int) ([]*model.ResourceTree, error) {
	// 获取子节点 ID Map
	nodeIdsMap := ts.getChildrenTreeNodeIds(ctx, nid)

	// 查询对应类型的资源
	resources := make([]*model.ResourceTree, 0)

	switch resourceType {
	case "ecs":
		ecs, err := ts.ecsDao.GetAll(ctx)
		if err != nil {
			ts.l.Error("GetEcsList failed", zap.Error(err))
			return nil, err
		}
		for _, e := range ecs {
			for _, n := range e.BindNodes {
				if _, ok := nodeIdsMap[n.ID]; ok {
					resources = append(resources, &e.ResourceTree)
					break
				}
			}
		}

	case "elb":
		elb, err := ts.elbDao.GetAll(ctx)
		if err != nil {
			ts.l.Error("GetElbList failed", zap.Error(err))
			return nil, err
		}
		for _, e := range elb {
			for _, n := range e.BindNodes {
				if _, ok := nodeIdsMap[n.ID]; ok {
					resources = append(resources, &e.ResourceTree)
					break
				}
			}
		}

	case "rds":
		rds, err := ts.rdsDao.GetAll(ctx)
		if err != nil {
			ts.l.Error("GetRdsList failed", zap.Error(err))
			return nil, err
		}
		for _, e := range rds {
			for _, n := range e.BindNodes {
				if _, ok := nodeIdsMap[n.ID]; ok {
					resources = append(resources, &e.ResourceTree)
					break
				}
			}
		}
	}

	// TODO 优化分页
	offset := (page - 1) * size
	if offset >= len(resources) {
		return nil, nil
	}

	end := offset + size
	if end > len(resources) {
		end = len(resources)
	}

	return resources[offset:end], nil
}

func (ts *treeService) BindEcs(ctx context.Context, ecsID int, treeNodeID int) error {
	ecs, err := ts.ecsDao.GetByIDNoPreload(ctx, ecsID)
	if err != nil {
		ts.l.Error("BindEcs 获取 ECS 失败", zap.Error(err))
		return err
	}

	node, err := ts.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		ts.l.Error("BindEcs 获取树节点失败", zap.Error(err))
		return err
	}

	return ts.ecsDao.AddBindNodes(ctx, ecs, node)
}

func (ts *treeService) BindElb(ctx context.Context, elbID int, treeNodeID int) error {
	elb, err := ts.elbDao.GetByIDNoPreload(ctx, elbID)
	if err != nil {
		ts.l.Error("BindElb 获取 ELB 失败", zap.Error(err))
		return err
	}

	node, err := ts.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		ts.l.Error("BindElb 获取树节点失败", zap.Error(err))
		return err
	}

	return ts.elbDao.AddBindNodes(ctx, elb, node)
}

func (ts *treeService) BindRds(ctx context.Context, rdsID int, treeNodeID int) error {
	elb, err := ts.rdsDao.GetByIDNoPreload(ctx, rdsID)
	if err != nil {
		ts.l.Error("BindRds 获取 RDS 失败", zap.Error(err))
		return err
	}

	node, err := ts.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		ts.l.Error("BindRds 获取树节点失败", zap.Error(err))
		return err
	}

	return ts.rdsDao.AddBindNodes(ctx, elb, node)
}

func (ts *treeService) UnBindEcs(ctx context.Context, ecsID int, treeNodeID int) error {
	ecs, err := ts.ecsDao.GetByIDNoPreload(ctx, ecsID)
	if err != nil {
		ts.l.Error("UnBindEcs 获取 ECS 失败", zap.Error(err))
		return err
	}

	node, err := ts.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		ts.l.Error("UnBindEcs 获取树节点失败", zap.Error(err))
		return err
	}

	return ts.ecsDao.RemoveBindNodes(ctx, ecs, node)
}

func (ts *treeService) UnBindElb(ctx context.Context, elbID int, treeNodeID int) error {
	elb, err := ts.elbDao.GetByIDNoPreload(ctx, elbID)
	if err != nil {
		ts.l.Error("UnBindElb 获取 ELB 失败", zap.Error(err))
		return err
	}

	node, err := ts.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		ts.l.Error("UnBindElb 获取树节点失败", zap.Error(err))
		return err
	}

	return ts.elbDao.RemoveBindNodes(ctx, elb, node)
}

func (ts *treeService) UnBindRds(ctx context.Context, rdsID int, treeNodeID int) error {
	rds, err := ts.rdsDao.GetByIDNoPreload(ctx, rdsID)
	if err != nil {
		ts.l.Error("UnBindRds 获取 RDS 失败", zap.Error(err))
		return err
	}

	node, err := ts.nodeDao.GetByIDNoPreload(ctx, treeNodeID)
	if err != nil {
		ts.l.Error("UnBindRds 获取树节点失败", zap.Error(err))
		return err
	}

	return ts.rdsDao.RemoveBindNodes(ctx, rds, node)
}

// fetchUsers 根据用户名列表获取用户，role 用于日志记录
func (ts *treeService) fetchUsers(ctx context.Context, userNames []string, role string) ([]*model.User, error) {
	// 预分配切片容量，减少内存重新分配次数
	users := make([]*model.User, 0, len(userNames))

	for _, userName := range userNames {
		user, err := ts.userDao.GetUserByUsername(ctx, userName)
		if err != nil {
			// 记录具体的错误信息，包括角色和用户名
			ts.l.Error(fmt.Sprintf("UpdateTreeNode 获取 %s 用户失败", role), zap.String("userName", userName), zap.Error(err))
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (ts *treeService) CreateEcsResource(ctx context.Context, resource *model.ResourceEcs) error {
	// 生成资源的唯一哈希值
	resource.Hash = generateResourceHash(resource)
	if err := ts.ecsDao.Create(ctx, resource); err != nil {
		ts.l.Error("CreateEcsResource 创建 ECS 资源失败", zap.Error(err))
		return err
	}

	return nil
}

// 生成资源哈希
func generateResourceHash(resource *model.ResourceEcs) string {
	// 假设根据资源的名称和IP地址生成唯一的哈希值，可以根据实际需求调整
	data := fmt.Sprintf("%s-%s", resource.InstanceName, resource.IpAddr)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (ts *treeService) UpdateEcsResource(ctx context.Context, resource *model.ResourceEcs) error {
	if err := ts.ecsDao.Update(ctx, resource); err != nil {
		ts.l.Error("UpdateEcsResource 更新 ECS 资源失败", zap.Error(err))
		return err
	}

	return nil
}

func (ts *treeService) DeleteEcsResource(ctx context.Context, id int) error {
	// 获取资源
	resource, err := ts.ecsDao.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrResourceNotFound) {
			ts.l.Warn("DeleteEcsResource: 资源未找到", zap.Int("id", id))
			return fmt.Errorf("资源未找到: %w", err)
		}
		ts.l.Error("DeleteEcsResource 获取 ECS 资源失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("获取 ECS 资源失败: %w", err)
	}

	// 检查资源是否被绑定
	if len(resource.BindNodes) > 0 {
		ts.l.Warn("DeleteEcsResource: 资源被绑定，无法删除", zap.Int("id", id), zap.Any("bindNodes", resource.BindNodes))
		return ErrResourceBound
	}

	// 删除资源
	if err := ts.ecsDao.Delete(ctx, id); err != nil {
		ts.l.Error("DeleteEcsResource 删除 ECS 资源失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("删除 ECS 资源失败: %w", err)
	}

	ts.l.Info("DeleteEcsResource 删除 ECS 资源成功", zap.Int("id", id))
	return nil
}

func (ts *treeService) GetAllResources(ctx context.Context, t string) ([]*model.ResourceTree, error) {
	return nil, nil
}
