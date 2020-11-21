package admin

import (
	"jiacrontab/models"
	"time"
)

// GetNodeList 获得任务节点列表
// 支持获得所属分组节点，指定分组节点（超级管理员）
func GetNodeList(ctx *myctx) {
	var (
		err      error
		nodeList []models.Node
		reqBody  GetNodeListReqParams
		count    int64
	)

	if err = ctx.Valid(&reqBody); err != nil {
		ctx.respParamError(err)
		return
	}

	if reqBody.QueryGroupID != ctx.claims.GroupID && !ctx.isSuper() {
		ctx.respNotAllowed()
		return
	}

	cfg := ctx.adm.getOpts()
	maxClientAliveInterval := -1 * cfg.App.MaxClientAliveInterval

	currentTime := time.Now().Add(time.Second * time.Duration(maxClientAliveInterval)).Format("2006-01-02 15:04:05")

	//失联列表更新为已断开状态
	models.DB().Unscoped().Model(&models.Node{}).Where("updated_at<?", currentTime).Updates(map[string]interface{}{
		"disabled": true,
	})

	txt := "%" + reqBody.SearchTxt + "%"
	switch reqBody.QueryStatus {
	case 1:
		err = models.DB().Preload("Group").Where("group_id=? and (name like ? or addr like ?) and disabled=?",
			reqBody.QueryGroupID, txt, txt, false).Offset((reqBody.Page - 1) * reqBody.Pagesize).Order("id desc").Limit(reqBody.Pagesize).Find(&nodeList).Error
		models.DB().Model(&models.Node{}).Where("group_id=? and (name like ? or addr like ?) and disabled=?",
			reqBody.QueryGroupID, txt, txt, false).Count(&count)
	case 2:
		err = models.DB().Preload("Group").Where("group_id=? and (name like ? or addr like ?) and disabled=?",
			reqBody.QueryGroupID, txt, txt, true).Offset((reqBody.Page - 1) * reqBody.Pagesize).Order("id desc").Limit(reqBody.Pagesize).Find(&nodeList).Error
		models.DB().Model(&models.Node{}).Where("group_id=? and (name like ? or addr like ?) and disabled=?", reqBody.QueryGroupID, txt, true).Count(&count)
	default:
		err = models.DB().Preload("Group").Where("group_id=? and (name like ? or addr like ?)",
			reqBody.QueryGroupID, txt, txt).Offset((reqBody.Page - 1) * reqBody.Pagesize).Order("id desc").Limit(reqBody.Pagesize).Find(&nodeList).Error
		models.DB().Model(&models.Node{}).Where("group_id=? and (name like ? or addr like ?)",
			reqBody.QueryGroupID, txt, txt).Count(&count)
	}

	if err != nil {
		ctx.respBasicError(err)
		return
	}

	ctx.respSucc("", map[string]interface{}{
		"list":     nodeList,
		"total":    count,
		"page":     reqBody.Page,
		"pagesize": reqBody.Pagesize,
	})
}

// DeleteNode 删除分组内节点
// 仅超级管理员有权限
func DeleteNode(ctx *myctx) {
	var (
		err     error
		reqBody DeleteNodeReqParams
		node    models.Node
	)

	if err = ctx.Valid(&reqBody); err != nil {
		ctx.respParamError(err)
	}

	// 普通用户不允许删除节点
	if !ctx.isSuper() {
		ctx.respNotAllowed()
		return
	}

	if err = node.Delete(reqBody.GroupID, reqBody.Addr); err != nil {
		ctx.respDBError(err)
		return
	}

	ctx.pubEvent(node.Addr, event_DelNodeDesc, models.EventSourceName(reqBody.Addr), reqBody)
	ctx.respSucc("", nil)
}

// GroupNode 超级管理员为node分组
// 分组不存在时自动创建分组
// copy超级管理员分组中的节点到新的分组
func GroupNode(ctx *myctx) {
	var (
		err     error
		reqBody GroupNodeReqParams
		node    models.Node
	)

	if !ctx.isSuper() {
		ctx.respNotAllowed()
		return
	}

	if err = ctx.Valid(&reqBody); err != nil {
		ctx.respParamError(err)
		return
	}

	if err = node.GroupNode(reqBody.Addr, reqBody.TargetGroupID,
		reqBody.TargetNodeName, reqBody.TargetGroupName); err != nil {
		ctx.respBasicError(err)
		return
	}

	ctx.pubEvent(node.Group.Name, event_GroupNode, models.EventSourceName(reqBody.Addr), reqBody)
	ctx.respSucc("", nil)
}

func UpdateNode(ctx *myctx) {}
