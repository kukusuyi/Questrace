import '../../shared/models/classification.dart';
import '../../shared/widgets/subject_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../core/network/api_client.dart';
import '../../core/network/api_exception.dart';
import '../../core/config/effective_api_base_url.dart';
import '../../core/storage/auth_session_repository.dart';
import '../../shared/widgets/latex_block.dart';
import '../../shared/widgets/remote_image_card.dart';
import '../../shared/widgets/tag_filter.dart';
import '../../shared/models/common_models.dart';
import '../question_list/question_repository.dart';
import '../dashboard/dashboard_repository.dart';

class ReviewPage extends ConsumerStatefulWidget {
  const ReviewPage({super.key, this.sessionId});
  final int? sessionId;
  @override
  ConsumerState<ReviewPage> createState() => _ReviewPageState();
}

class _ReviewPageState extends ConsumerState<ReviewPage> {
  final subject = TextEditingController(), note = TextEditingController();
  Classification classification = const Classification(status: '');
  String chapter = '';
  List<int> tags = [];
  String mastery = '';
  int count = 10;
  Map<String, dynamic>? session, summary;
  List<dynamic> history = [];
  bool loading = true, busy = false, revealed = false;
  String? error;
  String submission = '';
  static const results = {'forgot': '不会', 'partial': '模糊', 'correct': '会了'};
  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    subject.dispose();
    note.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      loading = true;
      error = null;
    });
    try {
      final dio = ref.read(apiClientProvider);
      if (widget.sessionId != null) {
        final r = await dio.get('/api/v1/reviews/sessions/${widget.sessionId}');
        session = Map<String, dynamic>.from(r.data as Map);
        _resetAnswer();
      } else {
        final responses = await Future.wait([
          dio.get('/api/v1/reviews/summary'),
          dio.get('/api/v1/reviews/history'),
        ]);
        summary = Map<String, dynamic>.from(responses[0].data as Map);
        history = responses[1].data as List;
      }
    } catch (e) {
      error = describeError(e);
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  List<dynamic> get items => session?['items'] as List? ?? [];
  Map<String, dynamic>? get current {
    for (final item in items) {
      if (item['result'] == '' && item['deleted'] != true) {
        return item as Map<String, dynamic>;
      }
    }
    return null;
  }

  void _resetAnswer() {
    revealed = false;
    note.clear();
    submission =
        '${DateTime.now().microsecondsSinceEpoch}-${widget.sessionId}-${current?['question_id']}';
  }

  Future<void> _create() async {
    setState(() => busy = true);
    try {
      final r = await ref
          .read(apiClientProvider)
          .post(
            '/api/v1/reviews/sessions',
            data: {
              'subject_id': classification.subjectId,
              'course_id': classification.courseId,
              'chapter': chapter,
              'tag_ids': tags,
              'mastery_status': mastery,
              'count': count,
            },
          );
      if (mounted) {
        final n = (r.data['items'] as List).length;
        if (n < count) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text('符合条件的题目共 $n 道，已全部加入')));
        }
        await context.push('/reviews/${r.data['id']}');
        if (mounted) await _load();
      }
    } catch (e) {
      if (mounted) setState(() => error = describeError(e));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  Future<void> _submit(String result) async {
    final q = current;
    if (q == null) return;
    setState(() => busy = true);
    try {
      final response = await ref
          .read(apiClientProvider)
          .post(
            '/api/v1/reviews/sessions/${widget.sessionId}/results',
            data: {
              'question_id': q['question_id'],
              'submission_id': submission,
              'result': result,
              'note': note.text,
            },
          );
      ref.invalidate(questionListProvider);
      ref.invalidate(dashboardSnapshotProvider);
      ref.invalidate(questionDetailProvider);
      if (mounted) {
        final due = DateTime.fromMillisecondsSinceEpoch(
          (response.data['due_at'] as int) * 1000,
        );
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              '${MasteryStatus.fromValue(response.data['mastery_status'] as String).label} · 下次复习：${due.toLocal().toString().substring(0, 10)}',
            ),
          ),
        );
        setState(() {
          q['result'] = result;
          _resetAnswer();
          error = null;
        });
      }
    } catch (e) {
      if (mounted) setState(() => error = describeError(e));
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  Future<void> _print() async {
    try {
      final base = ref.read(effectiveApiBaseUrlProvider),
          token = ref.read(authSessionRepositoryProvider).readToken();
      final uri = Uri.parse('$base/api/v1/wrong-questions/export/print')
          .replace(
            queryParameters: {
              'question_ids': items
                  .where((i) => i['deleted'] != true)
                  .map((i) => i['question_id'])
                  .join(','),
              'export_mode': 'questions_only',
              'access_token': token ?? '',
            },
          );
      if (!await launchUrl(uri, mode: LaunchMode.externalApplication)) {
        throw Exception('无法打开打印页');
      }
    } catch (e) {
      if (mounted) setState(() => error = describeError(e));
    }
  }

  Widget panel(List<Widget> children) => Card(
    child: Padding(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: children,
      ),
    ),
  );
  @override
  Widget build(BuildContext context) {
    final q = current;
    return Scaffold(
      appBar: AppBar(
        title: const Text('复习'),
        actions: [
          IconButton(
            onPressed: busy ? null : _load,
            tooltip: '刷新复习',
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                if (error != null)
                  Padding(
                    padding: const EdgeInsets.all(12),
                    child: Text(
                      error!,
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.error,
                      ),
                    ),
                  ),
                if (session != null) ...[
                  panel([
                    Text(
                      '已完成 ${items.where((i) => i['result'] != '' || i['deleted'] == true).length} / ${items.length} 题 · 进度自动保存',
                    ),
                    TextButton(
                      onPressed: _print,
                      child: const Text('导出本卷 PDF'),
                    ),
                  ]),
                  if (q != null)
                    panel([
                      Text(
                        Classification.fromJson(q).label(
                          q['subject'] as String? ?? '',
                          q['chapter'] as String? ?? '',
                        ),
                      ),
                      if (Classification.fromJson(q).statusLabel.isNotEmpty)
                        Text(Classification.fromJson(q).statusLabel),
                      LatexBlock(q['question_core'] as String),
                      if ((q['source_image_url'] as String).isNotEmpty)
                        RemoteImageCard(
                          imageUrl: q['source_image_url'] as String,
                        ),
                      const SizedBox(height: 16),
                      if (!revealed)
                        FilledButton(
                          onPressed: () => setState(() => revealed = true),
                          child: const Text('完成思考，查看答案'),
                        )
                      else ...[
                        const Text('答案与解析'),
                        LatexBlock(
                          (q['standard_solution'] as String).isEmpty
                              ? '暂无标准答案，请结合自己的解题过程自评。'
                              : q['standard_solution'] as String,
                        ),
                        const SizedBox(height: 12),
                        TextField(
                          controller: note,
                          maxLines: 3,
                          maxLength: 10000,
                          decoration: const InputDecoration(
                            labelText: '复习笔记（可选）',
                          ),
                        ),
                        const SizedBox(height: 12),
                        Wrap(
                          spacing: 8,
                          children: [
                            for (final r in results.entries)
                              FilledButton.tonal(
                                onPressed: busy ? null : () => _submit(r.key),
                                child: Text(r.value),
                              ),
                          ],
                        ),
                      ],
                    ])
                  else
                    panel([
                      const Text('本次练习已完成，下次复习已安排。'),
                      FilledButton(
                        onPressed: () => context.pop(),
                        child: const Text('返回复习'),
                      ),
                    ]),
                ] else ...[
                  panel([
                    Text(
                      '待复习 ${summary?['due'] ?? 0} 题',
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const Text('优先到期题目，再补未掌握和学习中的题目。连续三次到期答对后标为已掌握。'),
                    SubjectPicker(
                      value: classification,
                      chapter: chapter,
                      filter: true,
                      onChanged: (v, c) => setState(() {
                        classification = v;
                        chapter = c;
                        tags = [];
                      }),
                    ),
                    const SizedBox(height: 12),
                    TagFilter(
                      subjectId: classification.subjectId,
                      courseId: classification.courseId,
                      selected: tags,
                      onChanged: (v) => setState(() => tags = v),
                    ),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<String>(
                      initialValue: mastery,
                      decoration: const InputDecoration(labelText: '掌握状态'),
                      items: [
                        const DropdownMenuItem(
                          value: '',
                          child: Text('到期及待掌握'),
                        ),
                        for (final m in MasteryStatus.values)
                          DropdownMenuItem(
                            value: m.value,
                            child: Text(m.label),
                          ),
                      ],
                      onChanged: (v) => setState(() => mastery = v ?? ''),
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      initialValue: '10',
                      keyboardType: TextInputType.number,
                      decoration: const InputDecoration(labelText: '题数（1–100）'),
                      onChanged: (v) => count = int.tryParse(v) ?? 0,
                    ),
                    const SizedBox(height: 16),
                    FilledButton(
                      onPressed: busy ? null : _create,
                      child: const Text('自动组卷并开始'),
                    ),
                  ]),
                  panel([
                    const Text('我的练习'),
                    if ((summary?['sessions'] as List? ?? []).isEmpty)
                      const Text('开始第一份练习吧'),
                    for (final s in summary?['sessions'] as List? ?? [])
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        title: Text(
                          '练习 ${s['id']} · ${s['done']} / ${s['total']} 题',
                        ),
                        subtitle: Text(
                          DateTime.fromMillisecondsSinceEpoch(
                            (s['created_at'] as int) * 1000,
                          ).toString().substring(0, 16),
                        ),
                        trailing: Text(s['done'] < s['total'] ? '继续' : '查看'),
                        onTap: () async {
                          await context.push('/reviews/${s['id']}');
                          if (mounted) _load();
                        },
                      ),
                  ]),
                  panel([
                    const Text('最近复习记录'),
                    if (history.isEmpty) const Text('还没有复习记录'),
                    for (final h in history)
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        title: Text(
                          '题目 ${h['question_id']} · ${results[h['result']] ?? h['result']}',
                        ),
                        subtitle: Text('${h['note'] ?? ''}'),
                        onTap: () =>
                            context.push('/questions/${h['question_id']}'),
                      ),
                  ]),
                ],
              ],
            ),
    );
  }
}
