import 'classification.dart';
import 'json_helpers.dart';
import 'question_models.dart';

class AnalyzeWrongQuestionPayload {
  const AnalyzeWrongQuestionPayload({
    this.classification = const Classification(),
    required this.questionJson,
    this.providerName,
    this.modelName,
    this.chapter,
    this.ocrContext,
  });

  final Classification classification;
  final String? providerName;
  final String? modelName;
  final String? chapter;
  final QuestionJson questionJson;
  final OcrContext? ocrContext;

  Map<String, dynamic> toJson() {
    return {
      'subject_id': classification.subjectId,
      'course_id': classification.courseId,
      if (providerName != null && providerName!.isNotEmpty)
        'provider_name': providerName,
      if (modelName != null && modelName!.isNotEmpty) 'model_name': modelName,
      if (chapter != null && chapter!.isNotEmpty) 'chapter': chapter,
      'question_json': questionJson.toJson(),
      if (ocrContext != null) 'ocr_context': ocrContext!.toJson(),
    };
  }
}

class AnalyzeWrongQuestionResponse {
  const AnalyzeWrongQuestionResponse({
    this.warnings = const [],
    this.suggestedSubject = '',
    this.classification = const Classification(),
    required this.chapter,
    required this.tags,
    required this.semanticSummary,
    required this.mistakeSummary,
  });

  final Classification classification;
  final List<String> warnings;
  final String suggestedSubject;
  final String chapter;
  final TagGroups tags;
  final String semanticSummary;
  final String mistakeSummary;

  factory AnalyzeWrongQuestionResponse.fromJson(Map<String, dynamic> json) {
    return AnalyzeWrongQuestionResponse(
      classification: Classification.fromJson(json),
      warnings: asStringList(json['warnings']),
      suggestedSubject: asString(json['suggested_subject']),
      chapter: asString(json['chapter']),
      tags: TagGroups.fromJson(asJsonMap(json['tags'])),
      semanticSummary: asString(json['semantic_summary']),
      mistakeSummary: asString(json['mistake_summary']),
    );
  }
}

class AIProviderItem {
  const AIProviderItem({
    required this.providerName,
    required this.providerType,
    required this.configuredModel,
  });

  final String providerName;
  final String providerType;
  final String configuredModel;

  factory AIProviderItem.fromJson(Map<String, dynamic> json) {
    return AIProviderItem(
      providerName: asString(json['provider_name']),
      providerType: asString(json['provider_type']),
      configuredModel: asString(json['configured_model']),
    );
  }
}

class AIProviderListResponse {
  const AIProviderListResponse({required this.list});

  final List<AIProviderItem> list;

  factory AIProviderListResponse.fromJson(Map<String, dynamic> json) {
    return AIProviderListResponse(
      list: asObjectList(json['list'], AIProviderItem.fromJson),
    );
  }
}

class AIProviderModelItem {
  const AIProviderModelItem({required this.modelName});

  final String modelName;

  factory AIProviderModelItem.fromJson(Map<String, dynamic> json) {
    return AIProviderModelItem(modelName: asString(json['model_name']));
  }
}

class AIProviderModelListResponse {
  const AIProviderModelListResponse({
    required this.providerName,
    required this.list,
  });

  final String providerName;
  final List<AIProviderModelItem> list;

  factory AIProviderModelListResponse.fromJson(Map<String, dynamic> json) {
    return AIProviderModelListResponse(
      providerName: asString(json['provider_name']),
      list: asObjectList(json['list'], AIProviderModelItem.fromJson),
    );
  }
}

class AIChapterListResponse {
  const AIChapterListResponse({required this.list});

  final List<String> list;

  factory AIChapterListResponse.fromJson(Map<String, dynamic> json) {
    return AIChapterListResponse(list: asStringList(json['list']));
  }
}
