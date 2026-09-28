import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/network/api_client.dart';
import '../models/classification.dart';
import '../../features/auth/auth_controller.dart';
import '../models/json_helpers.dart';
import 'subject_suggestion.dart';

final subjectCatalogProvider = FutureProvider.autoDispose<List<SubjectItem>>((
  ref,
) async {
  ref.watch(authControllerProvider.select((state) => state.session));
  final r = await ref
      .watch(apiClientProvider)
      .get<Map<String, dynamic>>('/api/v1/subjects');
  return asObjectList(r.data?['list'], SubjectItem.fromJson);
});

class SubjectPicker extends ConsumerWidget {
  const SubjectPicker({
    super.key,
    required this.value,
    required this.chapter,
    required this.onChanged,
    this.filter = false,
    this.suggestedSubject = '',
    this.onSuggestionDismissed,
    this.allowAnalysisConfirmation = true,
  });
  final Classification value;
  final String chapter;
  final bool filter;
  final String suggestedSubject;
  final VoidCallback? onSuggestionDismissed;
  final bool allowAnalysisConfirmation;
  final void Function(Classification, String) onChanged;
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final catalog = ref.watch(subjectCatalogProvider);
    return catalog.when(
      loading: () => const LinearProgressIndicator(),
      error: (e, s) => TextButton(
        onPressed: () => ref.invalidate(subjectCatalogProvider),
        child: const Text('学科加载失败，点击重试'),
      ),
      data: (items) {
        SubjectItem? subject;
        for (final x in items) {
          if (x.id == value.subjectId) subject = x;
        }
        final selected = subject;
        SubjectCourse? course;
        for (final c in selected?.courses ?? <SubjectCourse>[]) {
          if (c.id == value.courseId) course = c;
        }
        final chapters = course?.chapters ?? selected?.chapters ?? <String>[];
        void select(String id) {
          SubjectItem? target;
          for (final s in items) {
            if (s.id == id) target = s;
          }
          onChanged(
            value.copyWith(
              subjectId: target?.id ?? '',
              subjectName: target?.name ?? '待分类',
              courseId: '',
              status: filter
                  ? value.status
                  : (target == null ? 'pending' : 'confirmed'),
              analysisStale:
                  !filter &&
                  (value.analysisStale ||
                      value.analysisConfirmed ||
                      value.revision > 0),
              analysisConfirmed: false,
            ),
            '',
          );
        }

        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (!filter &&
                value.subjectId.isEmpty &&
                suggestedSubject.isNotEmpty)
              SubjectSuggestion(
                name: suggestedSubject,
                onDismissed: onSuggestionDismissed ?? () {},
                onConfirmed: (subject) {
                  ref.invalidate(subjectCatalogProvider);
                  onChanged(
                    value.copyWith(
                      subjectId: subject.id,
                      subjectName: subject.name,
                      courseId: '',
                      status: 'confirmed',
                      analysisStale:
                          value.analysisStale ||
                          value.analysisConfirmed ||
                          value.revision > 0,
                      analysisConfirmed: false,
                    ),
                    '',
                  );
                },
              )
            else
              DropdownMenu<String>(
                key: ValueKey('subject-${value.subjectId}'),
                initialSelection: selected?.id ?? '',
                expandedInsets: EdgeInsets.zero,
                enableFilter: true,
                label: const Text('学科'),
                dropdownMenuEntries: [
                  DropdownMenuEntry(
                    value: '',
                    label: filter ? '全部学科' : '自动识别／待分类',
                  ),
                  for (final s in items.where(
                    (s) => s.recommended || s.id == value.subjectId,
                  ))
                    DropdownMenuEntry(value: s.id, label: s.name),
                ],
                onSelected: (id) {
                  if (id != null) select(id);
                },
              ),
            if (!filter) ...[
              const SizedBox(height: 8),
              Text(
                items.any(
                      (s) => s.recommended && s.educationStage == 'highschool',
                    )
                    ? '高中阶段 · 可添加语文、英语等其他科目'
                    : '大学阶段 · 支持材料、机械等任意专业科目',
              ),
            ],
            if (!filter)
              Align(
                alignment: Alignment.centerRight,
                child: TextButton.icon(
                  icon: const Icon(Icons.add),
                  label: const Text('添加任意学科'),
                  onPressed: () async {
                    String name = '';
                    final ok = await showDialog<bool>(
                      context: context,
                      builder: (ctx) => AlertDialog(
                        title: const Text('创建学科'),
                        content: TextField(
                          autofocus: true,
                          maxLength: 64,
                          decoration: const InputDecoration(
                            labelText: '学科名称',
                            hintText: '例如材料力学、机械原理',
                          ),
                          onChanged: (v) => name = v,
                        ),
                        actions: [
                          TextButton(
                            onPressed: () => Navigator.pop(ctx, false),
                            child: const Text('取消'),
                          ),
                          FilledButton(
                            onPressed: () => Navigator.pop(ctx, true),
                            child: const Text('创建'),
                          ),
                        ],
                      ),
                    );
                    if (ok != true || name.trim().isEmpty || !context.mounted) {
                      return;
                    }
                    try {
                      final r = await ref
                          .read(apiClientProvider)
                          .post<Map<String, dynamic>>(
                            '/api/v1/subjects',
                            data: {'name': name.trim()},
                          );
                      final s = SubjectItem.fromJson(r.data ?? {});
                      ref.invalidate(subjectCatalogProvider);
                      onChanged(
                        value.copyWith(
                          subjectId: s.id,
                          subjectName: s.name,
                          courseId: '',
                          status: 'confirmed',
                          analysisStale:
                              value.analysisStale ||
                              value.analysisConfirmed ||
                              value.revision > 0,
                          analysisConfirmed: false,
                        ),
                        '',
                      );
                    } catch (e) {
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('创建学科失败，请重试')),
                        );
                      }
                    }
                  },
                ),
              ),
            if (selected?.courses.isNotEmpty == true) ...[
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                key: ValueKey('course-${value.subjectId}-${value.courseId}'),
                initialValue: course?.id ?? '',
                isExpanded: true,
                decoration: const InputDecoration(labelText: '课程'),
                items: [
                  DropdownMenuItem(
                    value: '',
                    child: Text(filter ? '全部课程' : '自动识别课程'),
                  ),
                  for (final c in selected!.courses)
                    DropdownMenuItem(value: c.id, child: Text(c.name)),
                ],
                onChanged: (v) => onChanged(
                  value.copyWith(
                    courseId: v ?? '',
                    analysisStale:
                        !filter &&
                        (value.analysisStale ||
                            value.analysisConfirmed ||
                            value.revision > 0),
                    analysisConfirmed: false,
                  ),
                  '',
                ),
              ),
            ],
            if (selected != null) ...[
              const SizedBox(height: 12),
              if (selected.specialized)
                DropdownButtonFormField<String>(
                  key: ValueKey(
                    'chapter-${value.subjectId}-${value.courseId}-$chapter',
                  ),
                  initialValue: chapters.contains(chapter) ? chapter : '',
                  isExpanded: true,
                  decoration: const InputDecoration(labelText: '章节'),
                  items: [
                    DropdownMenuItem(
                      value: '',
                      child: Text(filter ? '全部章节' : '自动识别章节'),
                    ),
                    for (final c in chapters)
                      DropdownMenuItem(value: c, child: Text(c)),
                  ],
                  onChanged: selected.courses.isNotEmpty && course == null
                      ? null
                      : (v) => onChanged(
                          value.copyWith(
                            analysisStale:
                                !filter &&
                                (value.analysisStale ||
                                    value.analysisConfirmed ||
                                    value.revision > 0),
                            analysisConfirmed: false,
                          ),
                          v ?? '',
                        ),
                )
              else
                TextFormField(
                  key: ValueKey('free-chapter-${value.subjectId}'),
                  initialValue: chapter,
                  decoration: const InputDecoration(labelText: '章节或主题（可选）'),
                  onChanged: (v) => onChanged(
                    value.copyWith(
                      analysisStale:
                          !filter &&
                          (value.analysisStale ||
                              value.analysisConfirmed ||
                              value.revision > 0),
                      analysisConfirmed: false,
                    ),
                    v,
                  ),
                ),
            ],
            if (!filter && allowAnalysisConfirmation && value.analysisStale)
              CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                title: const Text('已复核标签和摘要，适用于当前学科'),
                value: value.analysisConfirmed,
                onChanged: (v) => onChanged(
                  value.copyWith(analysisConfirmed: v ?? false),
                  chapter,
                ),
              ),
          ],
        );
      },
    );
  }
}
