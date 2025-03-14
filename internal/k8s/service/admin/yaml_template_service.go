package admin

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
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	yamlTask "k8s.io/apimachinery/pkg/util/yaml"
)

type YamlTemplateService interface {
	// GetYamlTemplateList 获取 YAML 模板列表
	GetYamlTemplateList(ctx context.Context) ([]*model.K8sYamlTemplate, error)
	// CreateYamlTemplate 创建 YAML 模板
	CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// UpdateYamlTemplate 更新 YAML 模板
	UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// DeleteYamlTemplate 删除 YAML 模板
	DeleteYamlTemplate(ctx context.Context, id int) error
}

type yamlTemplateService struct {
	yamlTemplateDao admin.YamlTemplateDAO
	yamlTaskDao     admin.YamlTaskDAO
	client          client.K8sClient
	l               *zap.Logger
}

func NewYamlTemplateService(yamlTemplateDao admin.YamlTemplateDAO, yamlTaskDao admin.YamlTaskDAO, client client.K8sClient, l *zap.Logger) YamlTemplateService {
	return &yamlTemplateService{
		yamlTemplateDao: yamlTemplateDao,
		yamlTaskDao:     yamlTaskDao,
		client:          client,
		l:               l,
	}
}

// GetYamlTemplateList 获取 YAML 模板列表
func (y *yamlTemplateService) GetYamlTemplateList(ctx context.Context) ([]*model.K8sYamlTemplate, error) {
	return y.yamlTemplateDao.ListAllYamlTemplates(ctx)
}

// CreateYamlTemplate 创建 YAML 模板
func (y *yamlTemplateService) CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 校验 YAML 格式是否正确
	if _, err := yamlTask.ToJSON([]byte(template.Content)); err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	return y.yamlTemplateDao.CreateYamlTemplate(ctx, template)
}

// UpdateYamlTemplate 更新 YAML 模板
func (y *yamlTemplateService) UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 校验 YAML 格式是否正确
	if _, err := yamlTask.ToJSON([]byte(template.Content)); err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	// 更新模板
	return y.yamlTemplateDao.UpdateYamlTemplate(ctx, template)
}

// DeleteYamlTemplate 删除 YAML 模板
func (y *yamlTemplateService) DeleteYamlTemplate(ctx context.Context, id int) error {
	// 检查是否有任务正在使用该模板
	tasks, err := y.yamlTaskDao.GetYamlTaskByTemplateID(ctx, id)
	if err != nil {
		return err
	}

	// 如果有任务使用该模板，返回错误
	if len(tasks) > 0 {
		taskNames := make([]string, len(tasks))
		for i, task := range tasks {
			taskNames[i] = task.Name
		}
		return fmt.Errorf("该模板正在被以下任务使用: %v, 删除失败", taskNames)
	}

	// 删除模板
	return y.yamlTemplateDao.DeleteYamlTemplate(ctx, id)
}
