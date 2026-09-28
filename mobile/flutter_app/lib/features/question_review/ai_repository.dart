import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/network/api_client.dart';
import '../../shared/models/ai_models.dart';
import '../../shared/models/question_models.dart';

final aiRepositoryProvider = Provider<AIRepository>((ref) {
  return AIRepository(ref.watch(apiClientProvider));
});

class AIRepository {
  AIRepository(this._dio);

  final Dio _dio;

  Future<AIProviderListResponse> listProviders() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/v1/ai/model-providers',
    );

    return AIProviderListResponse.fromJson(
      response.data ?? <String, dynamic>{},
    );
  }

  Future<AIProviderModelListResponse> listProviderModels(
    String providerName,
  ) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/v1/ai/model-providers/${Uri.encodeComponent(providerName)}/models',
    );

    return AIProviderModelListResponse.fromJson(
      response.data ?? <String, dynamic>{},
    );
  }

  Future<AIChapterListResponse> listChapters({
    String subjectId = '',
    String courseId = '',
  }) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/v1/ai/chapters',
      queryParameters: {'subject_id': subjectId, 'course_id': courseId},
    );

    return AIChapterListResponse.fromJson(response.data ?? <String, dynamic>{});
  }

  Future<RecognizedWrongQuestion> recognizeWrongQuestion({
    required String imageUrl,
    required int imageId,
    String purpose = 'question',
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/v1/ocr/wrong-question-json',
      data: {'image_url': imageUrl, 'image_id': imageId, 'purpose': purpose},
      options: Options(
        sendTimeout: const Duration(minutes: 5),
        receiveTimeout: const Duration(minutes: 5),
      ),
    );

    return RecognizedWrongQuestion.fromJson(
      response.data ?? <String, dynamic>{},
    );
  }

  Future<AnalyzeWrongQuestionResponse> analyzeWrongQuestion(
    AnalyzeWrongQuestionPayload payload,
  ) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/v1/ai/analyze-wrong-question',
      data: payload.toJson(),
    );

    return AnalyzeWrongQuestionResponse.fromJson(
      response.data ?? <String, dynamic>{},
    );
  }
}
