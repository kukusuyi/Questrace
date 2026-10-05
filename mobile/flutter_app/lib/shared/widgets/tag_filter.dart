import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../features/tag_manage/tag_repository.dart';
import '../models/tag_models.dart';

final filterTagsProvider = FutureProvider.autoDispose
    .family<TagListResponse, ({String subject, String course})>(
      (ref, scope) => ref
          .watch(tagRepositoryProvider)
          .listTags(subjectId: scope.subject, courseId: scope.course),
    );

class TagFilter extends ConsumerStatefulWidget {
  const TagFilter({
    super.key,
    this.subjectId = '',
    this.courseId = '',
    required this.selected,
    required this.onChanged,
  });
  final String subjectId, courseId;
  final List<int> selected;
  final ValueChanged<List<int>> onChanged;
  @override
  ConsumerState<TagFilter> createState() => _TagFilterState();
}

class _TagFilterState extends ConsumerState<TagFilter> {
  String keyword = '';
  @override
  Widget build(BuildContext context) {
    final tags = ref.watch(
      filterTagsProvider((subject: widget.subjectId, course: widget.courseId)),
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextField(
          decoration: const InputDecoration(labelText: '搜索标签'),
          onChanged: (v) => setState(() => keyword = v.toLowerCase()),
        ),
        tags.when(
          loading: () => const LinearProgressIndicator(),
          error: (e, _) => TextButton(
            onPressed: () => ref.invalidate(filterTagsProvider),
            child: const Text('标签加载失败，点击重试'),
          ),
          data: (data) => Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              for (final type in TagType.values)
                if (data.list.any(
                  (t) =>
                      t.tagType == type &&
                      t.tagName.toLowerCase().contains(keyword),
                )) ...[
                  Padding(
                    padding: const EdgeInsets.only(top: 12),
                    child: Text(type.label),
                  ),
                  Wrap(
                    spacing: 8,
                    children: [
                      for (final tag in data.list.where(
                        (t) =>
                            t.tagType == type &&
                            t.tagName.toLowerCase().contains(keyword),
                      ))
                        FilterChip(
                          label: Text(tag.tagName),
                          selected: widget.selected.contains(tag.tagId),
                          onSelected: (v) => widget.onChanged(
                            v
                                ? [...widget.selected, tag.tagId]
                                : widget.selected
                                      .where((id) => id != tag.tagId)
                                      .toList(),
                          ),
                        ),
                    ],
                  ),
                ],
            ],
          ),
        ),
      ],
    );
  }
}
