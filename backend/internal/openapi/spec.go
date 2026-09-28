package openapi

func BuildSpec() map[string]any {
	paths := buildPaths()

	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "题迹 Questrace Backend API",
			"description": "错题本后端接口文档。当前文档与已初始化的后端骨架保持一致，便于前端联调和后续逐步补全持久化实现。",
			"version":     "2.0.0",
		},
		"servers": []map[string]any{
			{"url": "/"},
		},
		"tags": []map[string]any{
			{"name": "System", "description": "系统与健康检查"},
			{"name": "Auth", "description": "用户注册与登录"},
			{"name": "User", "description": "用户信息"},
			{"name": "Dashboard", "description": "仪表盘统计"},
			{"name": "Tag", "description": "标签管理"},
			{"name": "File", "description": "图片上传"},
			{"name": "OCR", "description": "图片转错题 JSON"},
			{"name": "AI", "description": "AI 分析"},
			{"name": "Question", "description": "错题管理与相似题查询"},
			{"name": "Docs", "description": "接口文档"},
		},
		"paths": paths,
		"components": map[string]any{
			"schemas":         schemas(),
			"securitySchemes": securitySchemes(),
		},
	}

	addLocalPaths(paths)
	addClassificationReviewPaths(paths)
	applySecurity(paths)

	return spec
}

func securitySchemes() map[string]any {
	return map[string]any{
		"bearerAuth": map[string]any{
			"type":         "http",
			"scheme":       "bearer",
			"bearerFormat": "JWT",
			"description":  "在 Authorization 头中传入 Bearer <JWT token>",
		},
	}
}

func basePaths() map[string]any {
	return map[string]any{
		"/healthz": map[string]any{
			"get": operation("System", "healthz", "健康检查", "返回服务健康状态。", nil, successNoEnvelope("text/plain", "ok")),
		},
		"/docs": map[string]any{
			"get": operation("Docs", "docsPage", "查看接口文档页", "返回内置 HTML 接口文档页面。", nil, successNoEnvelope("text/html", "<html>...</html>")),
		},
		"/docs/openapi.json": map[string]any{
			"get": operation("Docs", "openapiJson", "获取 OpenAPI JSON", "返回当前后端服务的 OpenAPI 规范。", nil, successNoEnvelope("application/json", map[string]any{"openapi": "3.0.3"})),
		},

		"/api/v1/auth/register": map[string]any{
			"post": operationWithBody(
				"Auth",
				"register",
				"用户注册",
				"注册新用户，用户名、密码、邮箱均为必填。注册成功后直接返回 JWT token。",
				nil,
				"#/components/schemas/RegisterRequest",
				successRef("#/components/schemas/RegisterResponse"),
			),
		},
		"/api/v1/auth/login": map[string]any{
			"post": operationWithBody(
				"Auth",
				"login",
				"用户登录",
				"使用用户名和密码登录，返回 JWT token。后续请求需在 Authorization 头中携带此 token。",
				nil,
				"#/components/schemas/LoginRequest",
				successRef("#/components/schemas/LoginResponse"),
			),
		},

		"/api/v1/users/me": map[string]any{
			"get": operation("User", "getUserMe", "获取当前用户", "根据 JWT token 返回当前登录用户的详细信息。", nil, successRef("#/components/schemas/UserMeResponse")),
		},
		"/api/v1/dashboard/summary": map[string]any{
			"get": operation(
				"Dashboard",
				"getDashboardSummary",
				"获取仪表盘摘要",
				"返回仪表盘首页使用的核心统计，包括总题量、今日新增、掌握状态分布、来源分布和活跃标签数。",
				nil,
				successRef("#/components/schemas/DashboardSummaryResponse"),
			),
		},
		"/api/v1/dashboard/recent": map[string]any{
			"get": operation(
				"Dashboard",
				"getDashboardRecentQuestions",
				"获取仪表盘最近错题",
				"按创建时间倒序返回最近录入的错题列表，默认 4 条。",
				[]map[string]any{
					queryParam("limit", "integer", "返回数量，默认 4，最大 12", false, nil),
				},
				successRef("#/components/schemas/DashboardRecentResponse"),
			),
		},
		"/api/v1/dashboard/tags": map[string]any{
			"get": operation(
				"Dashboard",
				"getDashboardTopTags",
				"获取仪表盘高频标签",
				"按使用次数返回知识点标签和错因标签的排行榜，默认各 6 条。",
				[]map[string]any{
					queryParam("limit", "integer", "每组返回数量，默认 6，最大 12", false, nil),
				},
				successRef("#/components/schemas/DashboardTagsResponse"),
			),
		},
		"/api/v1/tags": map[string]any{
			"get": operation(
				"Tag",
				"listTags",
				"获取标签列表",
				"按标签类型和关键词筛选启用中的标签。",
				[]map[string]any{
					queryParam("tag_type", "string", "标签类型", false, enumValues("knowledge_point", "problem_type", "method", "mistake_reason")),
					queryParam("keyword", "string", "标签关键词", false, nil),
				},
				successRef("#/components/schemas/TagListResponse"),
			),
			"post": operationWithBody(
				"Tag",
				"createTag",
				"创建标签",
				"创建或激活一个标签定义。",
				nil,
				"#/components/schemas/CreateTagRequest",
				successRef("#/components/schemas/TagItem"),
			),
		},
		"/api/v1/tags/{tagID}": map[string]any{
			"delete": operation(
				"Tag",
				"deleteTag",
				"删除标签",
				"软删除标签，用户侧默认不再显示。",
				[]map[string]any{
					pathParam("tagID", "integer", "标签 ID"),
				},
				successRef("#/components/schemas/DeleteTagResponse"),
			),
		},
		"/api/v1/files/images": map[string]any{
			"post": multipartOperation(
				"File",
				"uploadWrongQuestionImage",
				"上传错题图片",
				"上传图片并返回 image_id 与 image_url，供 OCR 和正式创建错题使用。",
				"#/components/schemas/FileUploadMultipartRequest",
				successRef("#/components/schemas/FileUploadResponse"),
			),
		},
		"/api/v1/ocr/wrong-question-json": map[string]any{
			"post": operationWithBody(
				"OCR",
				"ocrWrongQuestionJSON",
				"OCR 生成错题 JSON",
				"根据上传后的图片信息生成统一错题 JSON。",
				nil,
				"#/components/schemas/OCRWrongQuestionRequest",
				successRef("#/components/schemas/OCRWrongQuestionResponse"),
			),
		},
		"/api/v1/ai/analyze-wrong-question": map[string]any{
			"post": operationWithBody(
				"AI",
				"analyzeWrongQuestion",
				"AI 分析错题",
				"根据前端选择的模型厂商和模型名称，结合标准错题 JSON 生成标签、题目语义摘要和错因摘要。",
				nil,
				"#/components/schemas/AnalyzeWrongQuestionRequest",
				successRef("#/components/schemas/AnalyzeWrongQuestionResponse"),
			),
		},
		"/api/v1/ai/chapters": map[string]any{
			"get": operation(
				"AI",
				"listAIChapters",
				"获取章节列表",
				"按学科与课程返回可选章节；未提供学科时返回空列表。",
				nil,
				successRef("#/components/schemas/AIChapterListResponse"),
			),
		},
		"/api/v1/ai/model-providers": map[string]any{
			"get": operation(
				"AI",
				"listAIModelProviders",
				"获取模型厂商列表",
				"返回 config.yaml 中配置的模型厂商列表，供前端选择。",
				nil,
				successRef("#/components/schemas/AIProviderListResponse"),
			),
		},
		"/api/v1/ai/model-providers/{providerName}/models": map[string]any{
			"get": operation(
				"AI",
				"listAIProviderModels",
				"获取厂商模型列表",
				"调用对应厂商的模型列表接口，返回当前可调用模型。",
				[]map[string]any{
					pathParam("providerName", "string", "模型厂商标识"),
				},
				successRef("#/components/schemas/AIProviderModelListResponse"),
			),
		},
		"/api/v1/wrong-questions": map[string]any{
			"get": operation(
				"Question",
				"listWrongQuestions",
				"获取错题列表",
				"分页查询错题，支持学科、章节、标签、掌握状态和难度过滤。",
				[]map[string]any{
					queryParam("page", "integer", "页码，默认 1", false, nil),
					queryParam("page_size", "integer", "每页数量，默认 20", false, nil),
					queryParam("subject", "string", "学科", false, nil),
					queryParam("chapter", "string", "章节", false, nil),
					queryParam("keyword", "string", "关键词", false, nil),
					queryParam("tag_ids", "string", "标签 ID，多个用逗号分隔", false, nil),
					queryParam("mastery_status", "string", "掌握状态", false, enumValues("unmastered", "learning", "mastered")),
					queryParam("difficulty_level", "integer", "难度 1-5", false, nil),
					queryParam("source_type", "string", "来源类型", false, enumValues("manual", "image", "import")),
				},
				successRef("#/components/schemas/WrongQuestionPageResponse"),
			),
			"post": operationWithBody(
				"Question",
				"createWrongQuestion",
				"创建错题",
				"创建正式错题，并写入标签和向量占位数据。",
				nil,
				"#/components/schemas/CreateWrongQuestionRequest",
				successRef("#/components/schemas/CreateWrongQuestionResponse"),
			),
		},
		"/api/v1/wrong-questions/export/print": map[string]any{
			"get": operation(
				"Question",
				"exportWrongQuestionsPrint",
				"导出错题打印页",
				"根据选中的错题 ID 生成浏览器可直接打印并另存为 PDF 的 HTML 页面。",
				[]map[string]any{
					queryParam("question_ids", "string", "错题 ID，多个用逗号分隔，按传入顺序导出", true, nil),
					queryParam("export_mode", "string", "仅题目 A4 每页两题；with_answers 为兼容旧客户端保留，现同样只导出题目", false, enumValues("with_answers", "questions_only")),
					queryParam("access_token", "string", "浏览器直开导出页时使用的 JWT", false, nil),
				},
				successNoEnvelope("text/html", "<html><body>错题导出打印页</body></html>"),
			),
		},
		"/api/v1/wrong-questions/{questionID}": map[string]any{
			"get": operation(
				"Question",
				"getWrongQuestionDetail",
				"获取错题详情",
				"根据错题 ID 获取详情。",
				[]map[string]any{
					pathParam("questionID", "integer", "错题 ID"),
				},
				successRef("#/components/schemas/QuestionDetail"),
			),
			"put": operationWithBody(
				"Question",
				"updateWrongQuestion",
				"更新错题",
				"更新错题主数据、标签和向量占位数据。",
				[]map[string]any{
					pathParam("questionID", "integer", "错题 ID"),
				},
				"#/components/schemas/UpdateWrongQuestionRequest",
				successRef("#/components/schemas/UpdateWrongQuestionResponse"),
			),
			"delete": operation(
				"Question",
				"deleteWrongQuestion",
				"删除错题",
				"软删除错题，并标记相关向量为 deleted。",
				[]map[string]any{
					pathParam("questionID", "integer", "错题 ID"),
				},
				successRef("#/components/schemas/DeleteWrongQuestionResponse"),
			),
		},
		"/api/v1/wrong-questions/{questionID}/similar": map[string]any{
			"post": operationWithBody(
				"Question",
				"similarWrongQuestions",
				"根据已保存错题查询相似题",
				"对指定错题进行近似检索，支持语义或错因向量类型。",
				[]map[string]any{
					pathParam("questionID", "integer", "错题 ID"),
				},
				"#/components/schemas/SimilarQuestionRequest",
				successRef("#/components/schemas/SimilarQuestionResponse"),
			),
		},
		"/api/v1/wrong-questions/similar-by-json": map[string]any{
			"post": operationWithBody(
				"Question",
				"similarWrongQuestionsByJSON",
				"根据临时 JSON 查询相似题",
				"不创建正式错题，只用于保存前预检索。",
				nil,
				"#/components/schemas/SimilarByJSONRequest",
				successRef("#/components/schemas/SimilarQuestionResponse"),
			),
		},
	}
}

func baseSchemas() map[string]any {
	return map[string]any{
		"APIResponse": objectSchema(
			field("code", map[string]any{"type": "integer", "example": 0}),
			field("message", map[string]any{"type": "string", "example": "success"}),
			field("data", map[string]any{"nullable": true}),
		),

		"RegisterRequest": objectSchemaRequired(
			[]string{"username", "password", "email"},
			field("username", map[string]any{"type": "string", "example": "zhangsan"}),
			field("password", map[string]any{"type": "string", "format": "password", "example": "123456"}),
			field("email", map[string]any{"type": "string", "format": "email", "example": "zhangsan@example.com"}),
		),
		"RegisterResponse": objectSchema(
			field("user_id", map[string]any{"type": "integer", "example": 1}),
			field("username", map[string]any{"type": "string", "example": "zhangsan"}),
			field("token", map[string]any{"type": "string", "example": "eyJhbGciOiJIUzI1NiIs..."}),
		),
		"LoginRequest": objectSchemaRequired(
			[]string{"username", "password"},
			field("username", map[string]any{"type": "string", "example": "zhangsan"}),
			field("password", map[string]any{"type": "string", "format": "password", "example": "123456"}),
		),
		"LoginResponse": objectSchema(
			field("user_id", map[string]any{"type": "integer", "example": 1}),
			field("username", map[string]any{"type": "string", "example": "zhangsan"}),
			field("token", map[string]any{"type": "string", "example": "eyJhbGciOiJIUzI1NiIs..."}),
		),

		"QuestionJSON": objectSchemaRequired(
			[]string{"question_core"},
			field("question_core", map[string]any{"type": "string", "example": "求 lim_{x→0} sinx/x"}),
			field("standard_solution", map[string]any{"type": "string", "example": "利用基本极限可得结果为 1"}),
			field("wrong_solution", map[string]any{"type": "string", "example": "误将 sinx/x 当作 0/0 直接代入"}),
		),
		"TagGroups": objectSchema(
			field("knowledge_points", stringArraySchema([]string{"极限", "函数"})),
			field("problem_type", stringArraySchema([]string{"求解题"})),
			field("method", stringArraySchema([]string{"基本极限"})),
			field("mistake_reason", stringArraySchema([]string{"概念混淆"})),
		),
		"UserMeResponse": objectSchema(
			field("user_id", map[string]any{"type": "integer", "example": 1}),
			field("username", map[string]any{"type": "string", "example": "default_user"}),
			field("email", map[string]any{"type": "string", "example": "default@example.com"}),
			field("created_at", map[string]any{"type": "string", "format": "date-time"}),
		),
		"DashboardDistributionItem": objectSchema(
			field("type", map[string]any{"type": "string", "example": "unmastered"}),
			field("count", map[string]any{"type": "integer", "example": 12}),
		),
		"DashboardSummaryResponse": objectSchema(
			field("total_questions", map[string]any{"type": "integer", "example": 128}),
			field("today_added", map[string]any{"type": "integer", "example": 5}),
			field("unmastered_count", map[string]any{"type": "integer", "example": 74}),
			field("image_bound_count", map[string]any{"type": "integer", "example": 51}),
			field("active_tag_count", map[string]any{"type": "integer", "example": 37}),
			field("mastery_distribution", map[string]any{
				"type":  "array",
				"items": refSchema("#/components/schemas/DashboardDistributionItem"),
			}),
			field("source_distribution", map[string]any{
				"type":  "array",
				"items": refSchema("#/components/schemas/DashboardDistributionItem"),
			}),
		),
		"DashboardRecentResponse": objectSchema(
			field("list", map[string]any{
				"type":  "array",
				"items": refSchema("#/components/schemas/WrongQuestionListItem"),
			}),
		),
		"DashboardTagsResponse": objectSchema(
			field("knowledge_points", map[string]any{
				"type":  "array",
				"items": refSchema("#/components/schemas/TagItem"),
			}),
			field("mistake_reasons", map[string]any{
				"type":  "array",
				"items": refSchema("#/components/schemas/TagItem"),
			}),
		),
		"TagItem": objectSchema(
			field("tag_id", map[string]any{"type": "integer", "example": 1}),
			field("tag_name", map[string]any{"type": "string", "example": "极限"}),
			field("tag_type", map[string]any{"type": "string", "example": "knowledge_point"}),
			field("usage_count", map[string]any{"type": "integer", "example": 3}),
			field("is_active", map[string]any{"type": "boolean", "example": true}),
		),
		"TagListResponse": objectSchema(
			field("list", map[string]any{
				"type":  "array",
				"items": refSchema("#/components/schemas/TagItem"),
			}),
		),
		"CreateTagRequest": objectSchemaRequired(
			[]string{"tag_name", "tag_type"},
			field("tag_name", map[string]any{"type": "string", "example": "极限"}),
			field("tag_type", map[string]any{"type": "string", "example": "knowledge_point", "enum": []string{"knowledge_point", "problem_type", "method", "mistake_reason"}}),
		),
		"DeleteTagResponse": objectSchema(
			field("tag_id", map[string]any{"type": "integer", "example": 1}),
			field("deleted", map[string]any{"type": "boolean", "example": true}),
		),
		"FileUploadMultipartRequest": objectSchemaRequired(
			[]string{"file"},
			field("file", map[string]any{"type": "string", "format": "binary"}),
			field("usage", map[string]any{"type": "string", "example": "wrong_question"}),
		),
		"FileUploadResponse": objectSchema(
			field("image_id", map[string]any{"type": "integer", "example": 1}),
			field("image_url", map[string]any{"type": "string", "example": "http://localhost:8080/static/wrong-question/1.png"}),
			field("file_name", map[string]any{"type": "string", "example": "sample.png"}),
			field("file_size", map[string]any{"type": "integer", "example": 20480}),
			field("mime_type", map[string]any{"type": "string", "example": "image/png"}),
		),
		"OCRWrongQuestionRequest": objectSchemaRequired(
			[]string{"image_url", "image_id"},
			field("image_url", map[string]any{"type": "string", "example": "http://localhost:8080/static/wrong-question/1.png"}),
			field("purpose", map[string]any{"type": "string", "enum": []string{"question", "solution"}, "default": "question", "description": "question 识别题目；solution 仅抄录答案步骤，不解题，返回 standard_solution"}),
			field("image_id", map[string]any{"type": "integer", "example": 1}),
		),
		"OCRWrongQuestionResponse": objectSchema(
			field("question_core", map[string]any{"type": "string"}),
			field("standard_solution", map[string]any{"type": "string"}),
			field("wrong_solution", map[string]any{"type": "string"}),
			field("ocr_confidence", map[string]any{"type": "string", "enum": []string{"high", "medium", "low"}}),
			field("uncertain_parts", stringArraySchema([]string{"请确认题目主干"})),
		),
		"OCRContext": objectSchema(
			field("ocr_confidence", map[string]any{"type": "string", "enum": []string{"high", "medium", "low"}}),
			field("uncertain_parts", stringArraySchema([]string{"分母字符不清晰"})),
		),
		"AnalyzeWrongQuestionRequest": objectSchemaRequired(
			[]string{"question_json"},
			field("provider_name", map[string]any{"type": "string", "example": "qwen"}),
			field("model_name", map[string]any{"type": "string", "example": "qwen3.8-flash"}),
			field("chapter", map[string]any{"type": "string", "example": "函数的极限和连续"}),
			field("question_json", refSchema("#/components/schemas/QuestionJSON")),
			field("ocr_context", refSchema("#/components/schemas/OCRContext")),
		),
		"AnalyzeWrongQuestionResponse": objectSchemaRequired(
			[]string{"chapter", "tags", "semantic_summary"},
			field("chapter", map[string]any{"type": "string", "example": "函数的极限和连续"}),
			field("tags", refSchema("#/components/schemas/TagGroups")),
			field("semantic_summary", map[string]any{"type": "string"}),
			field("mistake_summary", map[string]any{"type": "string"}),
		),
		"AIProviderItem": objectSchema(
			field("provider_name", map[string]any{"type": "string", "example": "qwen"}),
			field("provider_type", map[string]any{"type": "string", "example": "qwen"}),
			field("configured_model", map[string]any{"type": "string", "example": "qwen3.8-flash"}),
		),
		"AIProviderListResponse": objectSchema(
			field("list", map[string]any{
				"type":  "array",
				"items": refSchema("#/components/schemas/AIProviderItem"),
			}),
		),
		"AIChapterListResponse": objectSchema(
			field("list", stringArraySchema([]string{"函数的极限和连续", "定积分"})),
		),
		"AIProviderModelItem": objectSchema(
			field("model_name", map[string]any{"type": "string", "example": "qwen3.8-flash"}),
		),
		"AIProviderModelListResponse": objectSchema(
			field("provider_name", map[string]any{"type": "string", "example": "qwen"}),
			field("list", map[string]any{
				"type":  "array",
				"items": refSchema("#/components/schemas/AIProviderModelItem"),
			}),
		),
		"CreateWrongQuestionRequest": objectSchemaRequired(
			[]string{"source_type", "subject", "question_json", "semantic_summary"},
			field("source_type", map[string]any{"type": "string", "enum": []string{"manual", "image", "import"}}),
			field("source_image_id", map[string]any{"type": "integer", "nullable": true}),
			field("source_image_url", map[string]any{"type": "string"}),
			field("subject", map[string]any{"type": "string", "example": "math"}),
			field("chapter", map[string]any{"type": "string", "example": "函数极限与连续"}),
			field("question_json", refSchema("#/components/schemas/QuestionJSON")),
			field("tags", refSchema("#/components/schemas/TagGroups")),
			field("semantic_summary", map[string]any{"type": "string"}),
			field("mistake_summary", map[string]any{"type": "string"}),
			field("difficulty_level", map[string]any{"type": "integer", "example": 3}),
			field("mastery_status", map[string]any{"type": "string", "enum": []string{"unmastered", "learning", "mastered"}}),
		),
		"CreateWrongQuestionResponse": objectSchema(
			field("question_id", map[string]any{"type": "integer", "example": 1}),
		),
		"WrongQuestionListItem": objectSchema(
			field("question_id", map[string]any{"type": "integer", "example": 1}),
			field("question_core", map[string]any{"type": "string"}),
			field("source_image_id", map[string]any{"type": "integer", "nullable": true}),
			field("source_image_url", map[string]any{"type": "string"}),
			field("subject", map[string]any{"type": "string"}),
			field("chapter", map[string]any{"type": "string"}),
			field("tags", refSchema("#/components/schemas/TagGroups")),
			field("difficulty_level", map[string]any{"type": "integer"}),
			field("mastery_status", map[string]any{"type": "string"}),
			field("created_at", map[string]any{"type": "string", "format": "date-time"}),
		),
		"WrongQuestionPageResponse": objectSchema(
			field("list", map[string]any{
				"type":  "array",
				"items": refSchema("#/components/schemas/WrongQuestionListItem"),
			}),
			field("total", map[string]any{"type": "integer", "example": 1}),
			field("page", map[string]any{"type": "integer", "example": 1}),
			field("page_size", map[string]any{"type": "integer", "example": 20}),
		),
		"QuestionDetail": objectSchema(
			field("question_id", map[string]any{"type": "integer", "example": 1}),
			field("question_core", map[string]any{"type": "string"}),
			field("standard_solution", map[string]any{"type": "string"}),
			field("wrong_solution", map[string]any{"type": "string"}),
			field("semantic_summary", map[string]any{"type": "string"}),
			field("mistake_summary", map[string]any{"type": "string"}),
			field("source_type", map[string]any{"type": "string"}),
			field("source_image_id", map[string]any{"type": "integer", "nullable": true}),
			field("source_image_url", map[string]any{"type": "string"}),
			field("subject", map[string]any{"type": "string"}),
			field("chapter", map[string]any{"type": "string"}),
			field("tags", refSchema("#/components/schemas/TagGroups")),
			field("difficulty_level", map[string]any{"type": "integer"}),
			field("mastery_status", map[string]any{"type": "string"}),
			field("created_at", map[string]any{"type": "string", "format": "date-time"}),
			field("updated_at", map[string]any{"type": "string", "format": "date-time"}),
		),
		"UpdateWrongQuestionRequest": objectSchemaRequired(
			[]string{"subject", "question_json", "semantic_summary"},
			field("question_json", refSchema("#/components/schemas/QuestionJSON")),
			field("subject", map[string]any{"type": "string"}),
			field("chapter", map[string]any{"type": "string"}),
			field("tags", refSchema("#/components/schemas/TagGroups")),
			field("source_image_id", map[string]any{"type": "integer", "nullable": true}),
			field("source_image_url", map[string]any{"type": "string"}),
			field("semantic_summary", map[string]any{"type": "string"}),
			field("mistake_summary", map[string]any{"type": "string"}),
			field("difficulty_level", map[string]any{"type": "integer"}),
			field("mastery_status", map[string]any{"type": "string", "enum": []string{"unmastered", "learning", "mastered"}}),
		),
		"UpdateWrongQuestionResponse": objectSchema(
			field("question_id", map[string]any{"type": "integer", "example": 1}),
			field("updated", map[string]any{"type": "boolean", "example": true}),
		),
		"DeleteWrongQuestionResponse": objectSchema(
			field("question_id", map[string]any{"type": "integer", "example": 1}),
			field("deleted", map[string]any{"type": "boolean", "example": true}),
		),
		"SimilarQuestionRequest": objectSchema(
			field("vector_type", map[string]any{"type": "string", "enum": []string{"semantic", "mistake"}, "example": "semantic"}),
			field("limit", map[string]any{"type": "integer", "example": 10}),
			field("use_tag_filter", map[string]any{"type": "boolean", "example": true}),
		),
		"SimilarByJSONRequest": objectSchemaRequired(
			[]string{"question_json"},
			field("question_json", refSchema("#/components/schemas/QuestionJSON")),
			field("tags", refSchema("#/components/schemas/TagGroups")),
			field("vector_type", map[string]any{"type": "string", "enum": []string{"semantic", "mistake"}}),
			field("limit", map[string]any{"type": "integer"}),
			field("use_tag_filter", map[string]any{"type": "boolean"}),
		),
		"SimilarQuestionItem": objectSchema(
			field("question_id", map[string]any{"type": "integer", "example": 2}),
			field("score", map[string]any{"type": "number", "format": "double", "example": 0.92}),
			field("similarity_type", map[string]any{"type": "string", "example": "hybrid"}),
			field("question_core", map[string]any{"type": "string"}),
			field("source_image_id", map[string]any{"type": "integer", "nullable": true}),
			field("source_image_url", map[string]any{"type": "string"}),
			field("matched_tags", stringArraySchema([]string{"极限"})),
			field("reason", map[string]any{"type": "string"}),
			field("tags", refSchema("#/components/schemas/TagGroups")),
		),
		"SimilarQuestionResponse": objectSchema(
			field("list", map[string]any{
				"type":  "array",
				"items": refSchema("#/components/schemas/SimilarQuestionItem"),
			}),
		),
	}
}

func operation(tag, operationID, summary, description string, parameters []map[string]any, response map[string]any) map[string]any {
	return map[string]any{
		"tags":        []string{tag},
		"operationId": operationID,
		"summary":     summary,
		"description": description,
		"parameters":  parameters,
		"responses": map[string]any{
			"200": response,
			"400": errorResponse("参数错误"),
			"401": errorResponse("认证失败"),
			"404": errorResponse("资源不存在"),
			"500": errorResponse("服务器内部错误"),
		},
	}
}

func operationWithBody(tag, operationID, summary, description string, parameters []map[string]any, requestRef string, response map[string]any) map[string]any {
	result := operation(tag, operationID, summary, description, parameters, response)
	result["requestBody"] = map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": refSchema(requestRef),
			},
		},
	}
	return result
}

func multipartOperation(tag, operationID, summary, description, requestRef string, response map[string]any) map[string]any {
	result := operation(tag, operationID, summary, description, nil, response)
	result["requestBody"] = map[string]any{
		"required": true,
		"content": map[string]any{
			"multipart/form-data": map[string]any{
				"schema": refSchema(requestRef),
			},
		},
	}
	return result
}

func successRef(dataRef string) map[string]any {
	return map[string]any{
		"description": "success",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": responseEnvelopeSchema(dataRef),
			},
		},
	}
}

func successNoEnvelope(contentType string, example any) map[string]any {
	return map[string]any{
		"description": "success",
		"content": map[string]any{
			contentType: map[string]any{
				"schema":  map[string]any{"type": "string"},
				"example": example,
			},
		},
	}
}

func errorResponse(message string) map[string]any {
	return map[string]any{
		"description": message,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"code":    map[string]any{"type": "integer", "example": 40001},
						"message": map[string]any{"type": "string", "example": message},
						"data":    map[string]any{"nullable": true},
					},
				},
			},
		},
	}
}

func responseEnvelopeSchema(dataRef string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code":    map[string]any{"type": "integer", "example": 0},
			"message": map[string]any{"type": "string", "example": "success"},
			"data":    refSchema(dataRef),
		},
	}
}

func refSchema(ref string) map[string]any {
	return map[string]any{"$ref": ref}
}

func objectSchema(fields ...map[string]any) map[string]any {
	properties := make(map[string]any, len(fields))
	for _, fieldItem := range fields {
		for name, schema := range fieldItem {
			properties[name] = schema
		}
	}

	return map[string]any{
		"type":       "object",
		"properties": properties,
	}
}

func objectSchemaRequired(required []string, fields ...map[string]any) map[string]any {
	schema := objectSchema(fields...)
	schema["required"] = required
	return schema
}

func field(name string, schema map[string]any) map[string]any {
	return map[string]any{name: schema}
}

func stringArraySchema(example []string) map[string]any {
	return map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "string",
		},
		"example": example,
	}
}

func queryParam(name, schemaType, description string, required bool, extra map[string]any) map[string]any {
	schema := map[string]any{"type": schemaType}
	for key, value := range extra {
		schema[key] = value
	}

	return map[string]any{
		"name":        name,
		"in":          "query",
		"required":    required,
		"description": description,
		"schema":      schema,
	}
}

func pathParam(name, schemaType, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "path",
		"required":    true,
		"description": description,
		"schema": map[string]any{
			"type": schemaType,
		},
	}
}

func enumValues(values ...string) map[string]any {
	return map[string]any{"enum": values}
}

var publicPathSet = map[string]bool{
	"/healthz":              true,
	"/docs":                 true,
	"/docs/openapi.json":    true,
	"/api/v1/auth/register": true,
	"/api/v1/auth/login":    true,
}

func applySecurity(paths map[string]any) {
	security := []map[string]any{
		{"bearerAuth": []any{}},
	}

	for path, methods := range paths {
		if publicPathSet[path] || path == "/api/v1/system/status" || path == "/api/v1/system/setup" || path == "/api/v1/auth/logout" {
			continue
		}
		for _, op := range methods.(map[string]any) {
			if o, ok := op.(map[string]any); ok {
				o["security"] = security
			}
		}
	}
}
