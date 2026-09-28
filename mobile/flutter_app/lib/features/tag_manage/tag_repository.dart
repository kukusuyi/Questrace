import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/network/api_client.dart';
import '../../shared/models/tag_models.dart';

final tagRepositoryProvider = Provider<TagRepository>((ref) {
  return TagRepository(ref.watch(apiClientProvider));
});

class TagRepository {
  TagRepository(this._dio);

  final Dio _dio;

  Future<TagListResponse> listTags({
    TagType? tagType,
    String keyword = '',
    String subjectId = '',
    String courseId = '',
  }) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/v1/tags',
      queryParameters: {
        if (subjectId.isNotEmpty) 'subject_id': subjectId,
        if (courseId.isNotEmpty) 'course_id': courseId,
        if (tagType != null) 'tag_type': tagType.value,
        if (keyword.trim().isNotEmpty) 'keyword': keyword.trim(),
      },
    );

    return TagListResponse.fromJson(response.data ?? <String, dynamic>{});
  }

  Future<TagItem> createTag({
    required String tagName,
    required TagType tagType,
  }) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/v1/tags',
      data: {'tag_name': tagName.trim(), 'tag_type': tagType.value},
    );

    return TagItem.fromJson(response.data ?? <String, dynamic>{});
  }

  Future<void> deleteTag(int tagId) async {
    await _dio.delete<Map<String, dynamic>>('/api/v1/tags/$tagId');
  }
}
