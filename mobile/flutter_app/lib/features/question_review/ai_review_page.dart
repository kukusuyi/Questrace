import '../../shared/widgets/subject_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/network/api_exception.dart';
import '../../shared/models/common_models.dart';
import '../../shared/models/question_models.dart';
import '../../shared/widgets/latex_block.dart';
import '../../shared/widgets/tag_groups_editor.dart';
import '../question_create/question_draft_controller.dart';
import 'question_flow_service.dart';

class AiReviewPage extends ConsumerStatefulWidget {
  const AiReviewPage({super.key});

  @override
  ConsumerState<AiReviewPage> createState() => _AiReviewPageState();
}

class _AiReviewPageState extends ConsumerState<AiReviewPage> {
  final _subjectController = TextEditingController();
  final _chapterController = TextEditingController();
  final _semanticSummaryController = TextEditingController();
  final _mistakeSummaryController = TextEditingController();
  final _knowledgePointsController = TextEditingController();
  final _problemTypeController = TextEditingController();
  final _methodController = TextEditingController();
  final _mistakeReasonController = TextEditingController();
  int _difficultyLevel = 3;
  MasteryStatus _masteryStatus = MasteryStatus.unmastered;
  bool _hydrated = false;
  bool _submitting = false;

  @override
  void dispose() {
    _subjectController.dispose();
    _chapterController.dispose();
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
    final draft = ref.watch(questionDraftControllerProvider);
    _hydrateIfNeeded(draft);

    if (draft == null) {
      return Scaffold(
        appBar: AppBar(title: const Text('AI 确认')),
        body: const Center(child: Text('当前没有可确认的 AI 草稿。')),
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text('AI 分析确认'),
        actions: [
          IconButton(
            icon: const Icon(Icons.delete_outline),
            tooltip: '放弃分析建议，返回编辑',
            onPressed: () => _confirmDiscard(),
          ),
        ],
      ),
      body: ListView(
        keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
        padding: const EdgeInsets.all(20),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '题目预览',
                    style: Theme.of(context).textTheme.titleLarge?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 12),
                  LatexBlock(draft.questionJson.questionCore),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                children: [
                  if (draft.warnings.isNotEmpty) Text(draft.warnings.join('；')),
                  SubjectPicker(
                    suggestedSubject: draft.suggestedSubject,
                    onSuggestionDismissed: () => ref
                        .read(questionDraftControllerProvider.notifier)
                        .dismissSuggestedSubject(),
                    value: draft.classification,
                    chapter: _chapterController.text,
                    onChanged: (v, c) {
                      _subjectController.text = v.subjectName;
                      _chapterController.text = c;
                      ref
                          .read(questionDraftControllerProvider.notifier)
                          .updateClassification(v, c);
                    },
                  ),
                  const SizedBox(height: 12),
                  TextFormField(
                    controller: _semanticSummaryController,
                    minLines: 2,
                    maxLines: 4,
                    decoration: const InputDecoration(labelText: '语义总结'),
                    onChanged: (_) => _persistDraft(),
                  ),
                  const SizedBox(height: 12),
                  TextFormField(
                    controller: _mistakeSummaryController,
                    minLines: 2,
                    maxLines: 4,
                    decoration: const InputDecoration(labelText: '错误总结'),
                    onChanged: (_) => _persistDraft(),
                  ),
                  const SizedBox(height: 16),
                  TagGroupsEditor(
                    knowledgePointsController: _knowledgePointsController,
                    problemTypeController: _problemTypeController,
                    methodController: _methodController,
                    mistakeReasonController: _mistakeReasonController,
                    onChanged: _persistDraft,
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
                      _persistDraft();
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
                        onChangeEnd: (_) => _persistDraft(),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _submitting ? null : _saveQuestion,
            child: Text(_submitting ? '保存中...' : '保存正式错题'),
          ),
        ],
      ),
    );
  }

  void _hydrateIfNeeded(QuestionDraft? draft) {
    if (_hydrated || draft == null) {
      return;
    }

    _subjectController.text = draft.subject;
    _chapterController.text = draft.chapter;
    _semanticSummaryController.text = draft.semanticSummary;
    _mistakeSummaryController.text = draft.mistakeSummary;
    _knowledgePointsController.text = draft.tags.knowledgePoints.join(', ');
    _problemTypeController.text = draft.tags.problemType.join(', ');
    _methodController.text = draft.tags.method.join(', ');
    _mistakeReasonController.text = draft.tags.mistakeReason.join(', ');
    _difficultyLevel = draft.difficultyLevel;
    _masteryStatus = draft.masteryStatus;
    _hydrated = true;
  }

  void _persistDraft() {
    ref
        .read(questionDraftControllerProvider.notifier)
        .updateReviewFields(
          subject: _subjectController.text.trim(),
          chapter: _chapterController.text.trim(),
          tags: TagGroups(
            knowledgePoints: _splitTags(_knowledgePointsController.text),
            problemType: _splitTags(_problemTypeController.text),
            method: _splitTags(_methodController.text),
            mistakeReason: _splitTags(_mistakeReasonController.text),
          ),
          semanticSummary: _semanticSummaryController.text.trim(),
          mistakeSummary: _mistakeSummaryController.text.trim(),
          difficultyLevel: _difficultyLevel,
          masteryStatus: _masteryStatus,
        );
  }

  Future<void> _saveQuestion() async {
    if (_subjectController.text.trim().isEmpty) {
      _showMessage('请先填写学科');
      return;
    }

    _persistDraft();

    setState(() {
      _submitting = true;
    });

    try {
      final questionId = await ref
          .read(questionFlowServiceProvider)
          .saveCurrentDraft();
      if (mounted) {
        context.go('/questions/$questionId');
      }
    } catch (error) {
      _showMessage(describeError(error));
    } finally {
      if (mounted) {
        setState(() {
          _submitting = false;
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

  Future<void> _confirmDiscard() async {
    await ref.read(questionDraftControllerProvider.notifier).discardAnalysis();
    if (mounted) context.go('/questions/create');
  }
}
