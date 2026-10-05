import '../../shared/models/classification.dart';
import '../../shared/widgets/subject_picker.dart';
import '../../shared/widgets/solution_ocr_button.dart';
import 'package:questrace_flutter/core/network/api_exception.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../shared/models/common_models.dart';
import '../../shared/models/question_models.dart';
import '../../shared/widgets/latex_review_field.dart';
import '../../shared/widgets/remote_image_card.dart';
import '../../shared/widgets/tag_groups_editor.dart';
import '../question_list/question_repository.dart';

class QuestionEditPage extends ConsumerStatefulWidget {
  const QuestionEditPage({super.key, required this.questionId});

  final int questionId;

  @override
  ConsumerState<QuestionEditPage> createState() => _QuestionEditPageState();
}

class _QuestionEditPageState extends ConsumerState<QuestionEditPage> {
  Classification _classification = const Classification();
  final _subjectController = TextEditingController();
  final _chapterController = TextEditingController();
  final _questionCoreController = TextEditingController();
  final _standardSolutionController = TextEditingController();
  final _wrongSolutionController = TextEditingController();
  final _semanticSummaryController = TextEditingController();
  final _mistakeSummaryController = TextEditingController();
  final _knowledgePointsController = TextEditingController();
  final _problemTypeController = TextEditingController();
  final _methodController = TextEditingController();
  final _mistakeReasonController = TextEditingController();

  int _difficultyLevel = 3;
  MasteryStatus _masteryStatus = MasteryStatus.unmastered;
  int? _hydratedQuestionId;
  bool _saving = false;

  @override
  void dispose() {
    _subjectController.dispose();
    _chapterController.dispose();
    _questionCoreController.dispose();
    _standardSolutionController.dispose();
    _wrongSolutionController.dispose();
    _semanticSummaryController.dispose();
    _mistakeSummaryController.dispose();
    _knowledgePointsController.dispose();
    _problemTypeController.dispose();
    _methodController.dispose();
    _mistakeReasonController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final detail = ref.watch(questionDetailProvider(widget.questionId));

    return Scaffold(
      appBar: AppBar(title: const Text('编辑错题')),
      body: detail.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, stackTrace) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Text(
              '错题编辑数据加载失败：${describeError(error)}',
              textAlign: TextAlign.center,
            ),
          ),
        ),
        data: (data) {
          _hydrateIfNeeded(data);

          return ListView(
            keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
            padding: const EdgeInsets.fromLTRB(20, 20, 20, 32),
            children: [
              if (data.sourceImageUrl.isNotEmpty) ...[
                RemoteImageCard(imageUrl: data.sourceImageUrl),
                const SizedBox(height: 16),
              ],
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(20),
                  child: Column(
                    children: [
                      SubjectPicker(
                        value: _classification,
                        chapter: _chapterController.text,
                        onChanged: (v, c) {
                          setState(() {
                            _classification = v;
                            _subjectController.text = v.subjectName;
                            _chapterController.text = c;
                          });
                        },
                      ),
                      const SizedBox(height: 12),
                      LatexReviewField(
                        title: '题目主干',
                        controller: _questionCoreController,
                        placeholder: '支持普通文本与 LaTeX 混排',
                        emptyPreviewText: '暂无题目主干',
                        onChanged: (_) {},
                      ),
                      const SizedBox(height: 12),
                      LatexReviewField(
                        title: '标准解',
                        controller: _standardSolutionController,
                        placeholder: '可以为空',
                        emptyPreviewText: '暂无标准解',
                        onChanged: (_) {},
                      ),
                      SolutionOcrButton(
                        controller: _standardSolutionController,
                      ),
                      const SizedBox(height: 12),
                      LatexReviewField(
                        title: '错误解',
                        controller: _wrongSolutionController,
                        placeholder: '可以为空，但建议保留原始错误过程',
                        emptyPreviewText: '暂无错误解',
                        onChanged: (_) {},
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _semanticSummaryController,
                        minLines: 2,
                        maxLines: 4,
                        decoration: const InputDecoration(labelText: '语义总结'),
                      ),
                      const SizedBox(height: 12),
                      TextFormField(
                        controller: _mistakeSummaryController,
                        minLines: 2,
                        maxLines: 4,
                        decoration: const InputDecoration(labelText: '错误总结'),
                      ),
                      const SizedBox(height: 16),
                      TagGroupsEditor(
                        knowledgePointsController: _knowledgePointsController,
                        problemTypeController: _problemTypeController,
                        methodController: _methodController,
                        mistakeReasonController: _mistakeReasonController,
                        onChanged: () {},
                      ),
                      const SizedBox(height: 16),
                      DropdownButtonFormField<MasteryStatus>(
                        initialValue: _masteryStatus,
                        decoration: const InputDecoration(labelText: '掌握状态'),
                        items: MasteryStatus.values
                            .map(
                              (item) => DropdownMenuItem(
                                value: item,
                                child: Text(item.label),
                              ),
                            )
                            .toList(),
                        onChanged: (value) {
                          if (value == null) {
                            return;
                          }
                          setState(() {
                            _masteryStatus = value;
                          });
                        },
                      ),
                      const SizedBox(height: 16),
                      Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('难度：$_difficultyLevel / 5'),
                          Slider(
                            min: 1,
                            max: 5,
                            divisions: 4,
                            value: _difficultyLevel.toDouble(),
                            onChanged: (value) {
                              setState(() {
                                _difficultyLevel = value.round();
                              });
                            },
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 16),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () => context.pop(),
                      child: const Text('取消'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton(
                      onPressed: _saving ? null : () => _save(data),
                      child: Text(_saving ? '保存中...' : '保存修改'),
                    ),
                  ),
                ],
              ),
            ],
          );
        },
      ),
    );
  }

  void _hydrateIfNeeded(QuestionDetail detail) {
    if (_hydratedQuestionId == detail.questionId) {
      return;
    }

    _classification = detail.classification;
    _subjectController.text = detail.subject;
    _chapterController.text = detail.chapter;
    _questionCoreController.text = detail.questionCore;
    _standardSolutionController.text = detail.standardSolution;
    _wrongSolutionController.text = detail.wrongSolution;
    _semanticSummaryController.text = detail.semanticSummary;
    _mistakeSummaryController.text = detail.mistakeSummary;
    _knowledgePointsController.text = detail.tags.knowledgePoints.join(', ');
    _problemTypeController.text = detail.tags.problemType.join(', ');
    _methodController.text = detail.tags.method.join(', ');
    _mistakeReasonController.text = detail.tags.mistakeReason.join(', ');
    _difficultyLevel = detail.difficultyLevel;
    _masteryStatus = detail.masteryStatus;
    _hydratedQuestionId = detail.questionId;
  }

  Future<void> _save(QuestionDetail current) async {
    if (_subjectController.text.trim().isEmpty) {
      _showMessage('请先填写学科');
      return;
    }
    if (_questionCoreController.text.trim().isEmpty) {
      _showMessage('请先填写题目主干');
      return;
    }
    if (_semanticSummaryController.text.trim().isEmpty) {
      _showMessage('请先填写语义总结');
      return;
    }

    setState(() {
      _saving = true;
    });

    try {
      await ref
          .read(questionRepositoryProvider)
          .updateQuestion(
            widget.questionId,
            UpdateWrongQuestionPayload(
              classification: _classification,
              questionJson: QuestionJson(
                questionCore: _questionCoreController.text.trim(),
                standardSolution: _standardSolutionController.text.trim(),
                wrongSolution: _wrongSolutionController.text.trim(),
              ),
              subject: _subjectController.text.trim(),
              chapter: _chapterController.text.trim(),
              tags: TagGroups(
                knowledgePoints: _splitTags(_knowledgePointsController.text),
                problemType: _splitTags(_problemTypeController.text),
                method: _splitTags(_methodController.text),
                mistakeReason: _splitTags(_mistakeReasonController.text),
              ),
              sourceImageId: current.sourceImageId,
              sourceImageUrl: current.sourceImageUrl,
              semanticSummary: _semanticSummaryController.text.trim(),
              mistakeSummary: _mistakeSummaryController.text.trim(),
              difficultyLevel: _difficultyLevel,
              masteryStatus: _masteryStatus,
            ),
          );

      ref.invalidate(questionDetailProvider(widget.questionId));
      ref.invalidate(questionListProvider(const ListQuestionFilter()));

      if (!mounted) {
        return;
      }

      _showMessage('错题更新成功');
      context.go('/questions/${widget.questionId}');
    } catch (error) {
      _showMessage('错题更新失败：${describeError(error)}');
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  List<String> _splitTags(String raw) {
    return raw
        .split(RegExp(r'[,，]'))
        .map((item) => item.trim())
        .where((item) => item.isNotEmpty)
        .toList();
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }
}
