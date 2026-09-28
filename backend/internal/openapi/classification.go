package openapi

func buildPaths() map[string]any {
	paths := basePaths()
	paths["/api/v1/users/me"].(map[string]any)["put"] = operationWithBody("User", "updateUserProfile", "修改个人学习阶段", "只修改当前账户的阶段；不改写已有题目、分类或向量。", nil, "#/components/schemas/UpdateUserProfile", successRef("#/components/schemas/UserMeResponse"))
	paths["/api/v1/subjects"] = map[string]any{
		"get":  operation("AI", "listSubjects", "学科目录", "内置学科及当前用户的自定义学科，含课程、章节和专用提示词能力。", nil, successRef("#/components/schemas/SubjectList")),
		"post": operationWithBody("AI", "createSubject", "创建或复用自定义学科", "名称去除首尾及重复空格、忽略大小写；按用户隔离。", nil, "#/components/schemas/CreateSubject", successRef("#/components/schemas/Subject")),
	}
	paths["/api/v1/wrong-questions/classification"] = map[string]any{"post": operationWithBody("Question", "reclassifyQuestions", "批量复核分类", "1至100题；按所有权及revision校验，冲突返回409，整批原子更新。原摘要和标签保留但分类改变后标记分析待更新。", nil, "#/components/schemas/ClassificationBatch", successRef("#/components/schemas/ClassificationBatchResult"))}
	for _, path := range []string{"/api/v1/ai/chapters", "/api/v1/tags", "/api/v1/wrong-questions"} {
		item := paths[path].(map[string]any)["get"].(map[string]any)
		params, _ := item["parameters"].([]map[string]any)
		params = append(params, queryParam("subject_id", "string", "稳定学科ID", false, nil), queryParam("course_id", "string", "408课程ID", false, nil))
		if path == "/api/v1/wrong-questions" {
			params = append(params, queryParam("classification_status", "string", "pending / legacy_pending / confirmed", false, nil))
		}
		item["parameters"] = params
	}
	return paths
}
func schemas() map[string]any {
	out := baseSchemas()
	stage := map[string]any{"type": "string", "enum": []string{"university", "highschool"}, "description": "大学 / 高中；历史账户升级默认大学，注册必须选择。"}
	out["UserMeResponse"].(map[string]any)["properties"].(map[string]any)["education_stage"] = stage
	reg := out["RegisterRequest"].(map[string]any)
	reg["properties"].(map[string]any)["education_stage"] = stage
	required, _ := reg["required"].([]string)
	reg["required"] = append(required, "education_stage")
	out["SystemSetup"] = objectSchemaRequired([]string{"token", "username", "email", "password", "education_stage"}, field("token", map[string]any{"type": "string"}), field("username", map[string]any{"type": "string"}), field("email", map[string]any{"type": "string"}), field("password", map[string]any{"type": "string", "minLength": 8}), field("education_stage", stage))
	out["UpdateUserProfile"] = objectSchemaRequired([]string{"education_stage"}, field("education_stage", stage))
	for _, name := range []string{"CreateWrongQuestionRequest", "UpdateWrongQuestionRequest", "WrongQuestionListItem", "QuestionDetail", "SimilarQuestionItem", "SimilarByJSONRequest", "AnalyzeWrongQuestionRequest", "AnalyzeWrongQuestionResponse"} {
		schema := out[name].(map[string]any)
		props := schema["properties"].(map[string]any)
		for _, f := range []string{"subject_id", "course_id", "classification_status"} {
			props[f] = map[string]any{"type": "string"}
		}
		if name != "AnalyzeWrongQuestionRequest" && name != "AnalyzeWrongQuestionResponse" {
			props["analysis_stale"] = map[string]any{"type": "boolean"}
			props["revision"] = map[string]any{"type": "integer"}
		}
		if name == "CreateWrongQuestionRequest" || name == "UpdateWrongQuestionRequest" {
			props["analysis_confirmed"] = map[string]any{"type": "boolean"}
			required, _ := schema["required"].([]string)
			next := []string{}
			for _, r := range required {
				if r != "subject" {
					next = append(next, r)
				}
			}
			schema["required"] = next
		}
	}
	for _, name := range []string{"SimilarQuestionRequest", "SimilarByJSONRequest"} {
		out[name].(map[string]any)["properties"].(map[string]any)["recall_scope"] = map[string]any{"type": "string", "enum": []string{"course", "subject"}, "default": "course", "description": "408默认同课程，subject仅扩大到整个408；始终不跨学科。"}
	}
	out["SimilarByJSONRequest"].(map[string]any)["properties"].(map[string]any)["chapter"] = map[string]any{"type": "string"}
	props := out["AnalyzeWrongQuestionResponse"].(map[string]any)["properties"].(map[string]any)
	props["subject"] = map[string]any{"type": "string"}
	props["suggested_subject"] = map[string]any{"type": "string"}
	props["warnings"] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	out["SimilarQuestionResponse"].(map[string]any)["properties"].(map[string]any)["status"] = map[string]any{"type": "string", "enum": []string{"ready", "no_matches", "indexing"}}
	out["SubjectCourse"] = objectSchema(field("id", map[string]any{"type": "string"}), field("name", map[string]any{"type": "string"}), field("chapters", map[string]any{"type": "array", "items": map[string]any{"type": "string"}}))
	out["Subject"] = objectSchema(field("id", map[string]any{"type": "string"}), field("name", map[string]any{"type": "string"}), field("specialized", map[string]any{"type": "boolean"}), field("courses", map[string]any{"type": "array", "items": refSchema("#/components/schemas/SubjectCourse")}), field("chapters", map[string]any{"type": "array", "items": map[string]any{"type": "string"}}))
	out["SubjectList"] = objectSchema(field("list", map[string]any{"type": "array", "items": refSchema("#/components/schemas/Subject")}))
	out["CreateSubject"] = objectSchemaRequired([]string{"name"}, field("name", map[string]any{"type": "string", "maxLength": 64}))
	out["ClassificationBatch"] = objectSchemaRequired([]string{"items", "subject_id"}, field("subject_id", map[string]any{"type": "string"}), field("course_id", map[string]any{"type": "string"}), field("chapter", map[string]any{"type": "string"}), field("items", map[string]any{"type": "array", "minItems": 1, "maxItems": 100, "items": objectSchemaRequired([]string{"question_id", "revision"}, field("question_id", map[string]any{"type": "integer"}), field("revision", map[string]any{"type": "integer"}))}))
	out["ClassificationBatchResult"] = objectSchema(field("updated", map[string]any{"type": "integer"}))
	out["ReviewCreate"] = objectSchema(field("subject_id", map[string]any{"type": "string"}), field("course_id", map[string]any{"type": "string"}), field("chapter", map[string]any{"type": "string"}), field("subject", map[string]any{"type": "string", "description": "历史名称筛选"}), field("tag_ids", map[string]any{"type": "array", "items": map[string]any{"type": "integer"}}), field("mastery_status", map[string]any{"type": "string"}), field("count", map[string]any{"type": "integer", "minimum": 1, "maximum": 100}))
	out["ReviewItem"] = objectSchema(field("question_id", map[string]any{"type": "integer"}), field("question_core", map[string]any{"type": "string"}), field("standard_solution", map[string]any{"type": "string"}), field("source_image_url", map[string]any{"type": "string"}), field("result", map[string]any{"type": "string"}), field("deleted", map[string]any{"type": "boolean"}), field("subject_id", map[string]any{"type": "string"}), field("subject", map[string]any{"type": "string"}), field("course_id", map[string]any{"type": "string"}), field("chapter", map[string]any{"type": "string"}), field("classification_status", map[string]any{"type": "string"}), field("analysis_stale", map[string]any{"type": "boolean"}))
	out["ReviewSession"] = objectSchema(field("id", map[string]any{"type": "integer"}), field("requested_count", map[string]any{"type": "integer"}), field("created_at", map[string]any{"type": "integer"}), field("items", map[string]any{"type": "array", "items": refSchema("#/components/schemas/ReviewItem")}))
	out["Subject"].(map[string]any)["properties"].(map[string]any)["education_stage"] = stage
	out["Subject"].(map[string]any)["properties"].(map[string]any)["recommended"] = map[string]any{"type": "boolean"}
	out["SubjectList"].(map[string]any)["properties"].(map[string]any)["education_stage"] = stage
	return out
}

func addClassificationReviewPaths(paths map[string]any) {
	for path, schema := range map[string]string{"/api/v1/system/setup": "SystemSetup", "/api/v1/admin/users": "RegisterRequest"} {
		paths[path].(map[string]any)["post"].(map[string]any)["requestBody"] = map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": refSchema("#/components/schemas/" + schema)}}}
	}
	paths["/api/v1/reviews/sessions"] = map[string]any{"post": operationWithBody("Question", "createReview", "按学科课程组卷", "按当前用户及指定学科、课程、章节筛选；分析待更新的题目不使用旧标签匹配。", nil, "#/components/schemas/ReviewCreate", successRef("#/components/schemas/ReviewSession"))}
	op := paths["/api/v1/reviews/sessions/{sessionID}"].(map[string]any)["get"].(map[string]any)
	op["responses"].(map[string]any)["200"] = successRef("#/components/schemas/ReviewSession")
}
