package api

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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TreeHandler struct {
	service    service.TreeService
	aliService service.AliResourceService
	l          *zap.Logger
}

func NewTreeHandler(service service.TreeService, l *zap.Logger, aliService service.AliResourceService) *TreeHandler {
	return &TreeHandler{
		service:    service,
		aliService: aliService,
		l:          l,
	}
}

func (t *TreeHandler) RegisterRouters(server *gin.Engine) {
	treeGroup := server.Group("/api/tree")

	// 树节点相关路由
	treeGroup.GET("/listTreeNode", t.ListTreeNode)
	treeGroup.GET("/selectTreeNode", t.SelectTreeNode)
	treeGroup.GET("/getTopTreeNode", t.GetTopTreeNode)
	treeGroup.GET("/listLeafTreeNode", t.ListLeafTreeNodes)
	treeGroup.POST("/createTreeNode", t.CreateTreeNode)
	treeGroup.DELETE("/deleteTreeNode/:id", t.DeleteTreeNode)
	treeGroup.GET("/getChildrenTreeNode/:pid", t.GetChildrenTreeNode)
	treeGroup.POST("/updateTreeNode", t.UpdateTreeNode)

	// ECS, ELB, RDS 资源相关路由
	treeGroup.GET("/getEcsUnbindList", t.GetEcsUnbindList)
	treeGroup.GET("/getEcsList", t.GetEcsList)
	treeGroup.GET("/getElbUnbindList", t.GetElbUnbindList)
	treeGroup.GET("/getElbList", t.GetElbList)
	treeGroup.GET("/getRdsUnbindList", t.GetRdsUnbindList)
	treeGroup.GET("/getRdsList", t.GetRdsList)
	treeGroup.GET("/getAllResourceByType", t.GetAllResourceByType)

	// 资源绑定相关路由
	treeGroup.POST("/bindEcs", t.BindEcs)
	treeGroup.POST("/bindElb", t.BindElb)
	treeGroup.POST("/bindRds", t.BindRds)
	treeGroup.POST("/unBindEcs", t.UnBindEcs)
	treeGroup.POST("/unBindElb", t.UnBindElb)
	treeGroup.POST("/unBindRds", t.UnBindRds)

	// 非公有云ECS资源CURD相关路由
	treeGroup.POST("/createEcsResource", t.CreateEcsResource)
	treeGroup.POST("/updateEcsResource", t.UpdateEcsResource)
	treeGroup.DELETE("/deleteEcsResource/:id", t.DeleteEcsResource)

	// 公有云ECS资源相关路由
	treeGroup.POST("/createAliResource", t.CreateAliEcsResource)
	treeGroup.POST("/updateAliResource", t.UpdateAliEcsResource)
	treeGroup.DELETE("/deleteAliResource/:id", t.DeleteAliEcsResource)
	treeGroup.GET("/getResourceStatus/:id", t.GetResourceStatus)
}

func (t *TreeHandler) ListTreeNode(ctx *gin.Context) {
	list, err := t.service.ListTreeNodes(ctx)

	if err != nil {
		t.l.Error("list tree nodes failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

func (t *TreeHandler) SelectTreeNode(ctx *gin.Context) {
	// 获取查询参数 "level" 和 "levelLt"，并设置默认值为 "0"
	levelStr := ctx.DefaultQuery("level", "0")
	levelLtStr := ctx.DefaultQuery("levelLt", "0")

	// 将字符串参数转换为整数，并处理转换错误
	level, err := strconv.Atoi(levelStr)
	if err != nil {
		t.l.Warn("无效的 level 参数", zap.String("level", levelStr), zap.Error(err))
		apiresponse.BadRequestError(ctx, "无效的 level 参数")
		return
	}

	levelLt, err := strconv.Atoi(levelLtStr)
	if err != nil {
		t.l.Warn("无效的 levelLt 参数", zap.String("levelLt", levelLtStr), zap.Error(err))
		apiresponse.BadRequestError(ctx, "无效的 levelLt 参数")
		return
	}

	// 调用服务层方法获取过滤后的树节点
	nodes, err := t.service.SelectTreeNode(ctx, level, levelLt)
	if err != nil {
		t.l.Error("SelectTreeNode 调用失败", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	// 返回成功响应，包含过滤后的树节点
	apiresponse.SuccessWithData(ctx, nodes)
}

func (t *TreeHandler) GetTopTreeNode(ctx *gin.Context) {
	nodes, err := t.service.GetTopTreeNode(ctx)
	if err != nil {
		t.l.Error("get top tree node failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, nodes)
}

func (t *TreeHandler) ListLeafTreeNodes(ctx *gin.Context) {
	list, err := t.service.ListLeafTreeNodes(ctx)
	if err != nil {
		t.l.Error("get all tree nodes failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

func (t *TreeHandler) CreateTreeNode(ctx *gin.Context) {
	var req model.TreeNode

	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.CreateTreeNode(ctx, &req); err != nil {
		t.l.Error("create tree node failed", zap.Error(err))
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) DeleteTreeNode(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "id不能为空")
		return
	}

	nodeId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "id必须为整数")
		return
	}

	if err := t.service.DeleteTreeNode(ctx, nodeId); err != nil {
		t.l.Error("delete tree node failed", zap.Error(err))
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) GetChildrenTreeNode(ctx *gin.Context) {
	pid := ctx.Param("pid")
	if pid == "" {
		apiresponse.BadRequestError(ctx, "pid不能为空")
		return
	}

	parentId, err := strconv.Atoi(pid)
	if err != nil {
		apiresponse.BadRequestError(ctx, "pid必须为整数")
		return
	}

	list, err := t.service.GetChildrenTreeNodes(ctx, parentId)
	if err != nil {
		t.l.Error("get children tree nodes failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

func (t *TreeHandler) UpdateTreeNode(ctx *gin.Context) {
	var req model.TreeNode
	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.UpdateTreeNode(ctx, &req); err != nil {
		t.l.Error("update tree node failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) GetEcsUnbindList(ctx *gin.Context) {
	ecs, err := t.service.GetEcsUnbindList(ctx)
	if err != nil {
		t.l.Error("get unbind ecs failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, ecs)
}

func (t *TreeHandler) GetEcsList(ctx *gin.Context) {
	ecs, err := t.service.GetEcsList(ctx)
	if err != nil {
		t.l.Error("get ecs list failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, ecs)
}

func (t *TreeHandler) GetElbUnbindList(ctx *gin.Context) {
	elb, err := t.service.GetElbUnbindList(ctx)
	if err != nil {
		t.l.Error("get unbind elb failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, elb)
}

func (t *TreeHandler) GetElbList(ctx *gin.Context) {
	elb, err := t.service.GetElbList(ctx)
	if err != nil {
		t.l.Error("get elb list failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, elb)
}

func (t *TreeHandler) GetRdsUnbindList(ctx *gin.Context) {
	rds, err := t.service.GetRdsUnbindList(ctx)
	if err != nil {
		t.l.Error("get unbind rds failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, rds)
}

func (t *TreeHandler) GetRdsList(ctx *gin.Context) {
	rds, err := t.service.GetRdsList(ctx)
	if err != nil {
		t.l.Error("get rds list failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, rds)
}

func (t *TreeHandler) GetAllResourceByType(ctx *gin.Context) {
	resourceType := ctx.Query("type")
	if resourceType == "" || (resourceType != "ecs" && resourceType != "elb" && resourceType != "rds") {
		apiresponse.BadRequestError(ctx, "resource type不能为空或不合法")
		return
	}

	nid := ctx.Query("nid")
	if nid == "" {
		apiresponse.BadRequestError(ctx, "nid不能为空")
		return
	}
	nodeId, err := strconv.Atoi(nid)
	if err != nil {
		apiresponse.BadRequestError(ctx, "nid必须为整数")
		return
	}

	p := ctx.DefaultQuery("page", "1")
	s := ctx.DefaultQuery("size", "10")
	page, err := strconv.Atoi(p)
	if err != nil {
		apiresponse.BadRequestError(ctx, "page必须为整数")
		return
	}
	size, err := strconv.Atoi(s)
	if err != nil {
		apiresponse.BadRequestError(ctx, "size必须为整数")
		return
	}

	resource, err := t.service.GetAllResourcesByType(ctx, nodeId, resourceType, page, size)
	if err != nil {
		t.l.Error("get all resource failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, resource)
}

func (t *TreeHandler) BindEcs(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.BindEcs(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		t.l.Error("bind ecs failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) BindElb(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.BindElb(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		t.l.Error("bind elb failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) BindRds(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.BindRds(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		t.l.Error("bind rds failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) UnBindEcs(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.UnBindEcs(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		t.l.Error("unbind ecs failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) UnBindElb(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.UnBindElb(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		t.l.Error("unbind elb failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) UnBindRds(ctx *gin.Context) {
	var req model.BindResourceReq

	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.UnBindRds(ctx, req.ResourceIds[0], req.NodeId); err != nil {
		t.l.Error("unbind rds failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) CreateEcsResource(ctx *gin.Context) {
	var req model.ResourceEcs

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.CreateEcsResource(ctx, &req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) UpdateEcsResource(ctx *gin.Context) {
	var req model.ResourceEcs

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.UpdateEcsResource(ctx, &req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) DeleteEcsResource(ctx *gin.Context) {
	id := ctx.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "id 非整数")
		return
	}

	if err := t.service.DeleteEcsResource(ctx, idInt); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) CreateAliEcsResource(ctx *gin.Context) {
	var req model.TerraformConfig

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	id, err := t.aliService.CreateResource(ctx, req)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, id)
}

func (t *TreeHandler) UpdateAliEcsResource(ctx *gin.Context) {
	var req model.TerraformConfig

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.aliService.UpdateResource(ctx, req.ID, req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) DeleteAliEcsResource(ctx *gin.Context) {
	id := ctx.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "id 非整数")
		return
	}

	if err := t.aliService.DeleteResource(ctx, idInt); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) GetResourceStatus(ctx *gin.Context) {
	id := ctx.Param("id")

	task, err := t.aliService.GetTaskStatus(ctx, id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, task)

}
