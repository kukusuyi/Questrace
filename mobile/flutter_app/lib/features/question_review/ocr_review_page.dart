import '../../shared/widgets/subject_picker.dart';
import '../../shared/widgets/solution_ocr_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/network/api_exception.dart';
import '../../shared/models/question_models.dart';
import '../../shared/widgets/analysis_picker.dart';
import '../../shared/models/common_models.dart';
import '../../shared/widgets/latex_review_field.dart';
import '../../shared/widgets/remote_image_card.dart';
import '../question_create/question_draft_controller.dart';
import 'question_flow_service.dart';

class OcrReviewPage extends ConsumerStatefulWidget {
  const OcrReviewPage({super.key});

  @override
  ConsumerState<OcrReviewPage> createState() => _OcrReviewPageState();
}

class _OcrReviewPageState extends ConsumerState<OcrReviewPage> {
  final _questionCoreController = TextEditingController();
  final _standardSolutionController = TextEditingController();
  final _wrongSolutionController = TextEditingController();
  bool _hydrated = false;
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
  }

  @override
  void dispose() {
    _questionCoreController.dispose();
    _standardSolutionController.dispose();
    _wrongSolutionController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final draft = ref.watch(questionDraftControllerProvider);
    _hydrateIfNeeded(draft);

    if (draft == null) {
      return Scaffold(
        appBar: AppBar(title: const Text('OCR 确认')),
        body: const Center(child: Text('当前没有可确认的 OCR 草稿。')),
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text('OCR 结果确认'),
        actions: [
          IconButton(
            icon: const Icon(Icons.delete_outline),
            tooltip: '丢弃草稿',
            onPressed: () => _confirmDiscard(),
          ),
        ],
      ),
      body: ListView(
        keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
        padding: const EdgeInsets.all(20),
        children: [
          if (draft.sourceImageUrl.isNotEmpty)
            RemoteImageCard(imageUrl: draft.sourceImageUrl),
          if (draft.sourceImageUrl.isNotEmpty) const SizedBox(height: 16),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'OCR 置信度：${draft.ocrContext?.ocrConfidence.label ?? '未知'}',
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  if ((draft.ocrContext?.uncertainParts ?? const <String>[])
                      .isNotEmpty) ...[
                    const SizedBox(height: 12),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: [
                        for (final item in draft.ocrContext!.uncertainParts)
                          Chip(label: Text(item)),
                      ],
                    ),
                  ],
                  const SizedBox(height: 16),
                  LatexReviewField(
                    title: '题目主干',
                    controller: _questionCoreController,
                    placeholder: '支持普通文本与 LaTeX 混排',
                    emptyPreviewText: '暂无题目主干',
                    onChanged: (_) => _persistDraft(),
                  ),
                  const SizedBox(height: 12),
                  LatexReviewField(
                    title: '标准解',
                    controller: _standardSolutionController,
                    placeholder: '可以为空',
                    emptyPreviewText: '暂无标准解',
                    onChanged: (_) => _persistDraft(),
                  ),
                  SolutionOcrButton(
                    controller: _standardSolutionController,
                    onApplied: _persistDraft,
                  ),
                  const SizedBox(height: 12),
                  LatexReviewField(
                    title: '错误解',
                    controller: _wrongSolutionController,
                    placeholder: '可以为空，但建议保留原始错误过程',
                    emptyPreviewText: '暂无错误解',
                    onChanged: (_) => _persistDraft(),
                  ),
                  const SizedBox(height: 12),
                  SubjectPicker(
                    suggestedSubject: draft.suggestedSubject,
                    onSuggestionDismissed: () => ref
                        .read(questionDraftControllerProvider.notifier)
                        .dismissSuggestedSubject(),
                    value: draft.classification,
                    chapter: draft.chapter,
                    onChanged: (v, c) => ref
                        .read(questionDraftControllerProvider.notifier)
                        .updateClassification(v, c),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _submitting ? null : _continueEditing,
            child: const Text('确认并整理内容'),
          ),
          const SizedBox(height: 12),
          FilledButton(
            onPressed: _submitting
                ? null
                : () async {
                    if (await showAnalysisPicker(context) && mounted) {
                      await _analyze();
                    }
                  },
            child: Text(_submitting ? '分析中...' : 'AI 辅助分析'),
          ),
        ],
      ),
    );
  }

  void _continueEditing() {
    final controller = ref.read(questionDraftControllerProvider.notifier);
    final draft = ref.read(questionDraftControllerProvider)!;
    controller.updateBasicFields(
      subject: draft.subject,
      chapter: draft.chapter,
      questionJson: QuestionJson(
        questionCore: _questionCoreController.text,
        standardSolution: _standardSolutionController.text,
        wrongSolution: _wrongSolutionController.text,
      ),
      flowMode: DraftFlowMode.manual,
      sourceType: SourceType.image,
    );
    controller.markStatus(DraftStatus.draft);
    context.go('/questions/create');
  }

  void _hydrateIfNeeded(QuestionDraft? draft) {
    if (_hydrated || draft == null) {
      return;
    }

    _questionCoreController.text = draft.questionJson.questionCore;
    _standardSolutionController.text = draft.questionJson.standardSolution;
    _wrongSolutionController.text = draft.questionJson.wrongSolution;
    _hydrated = true;
  }

  void _persistDraft() {
    final draft = ref.read(questionDraftControllerProvider);
    if (draft == null) {
      return;
    }

    ref
        .read(questionDraftControllerProvider.notifier)
        .updateBasicFields(
          subject: draft.subject,
          chapter: draft.chapter,
          questionJson: QuestionJson(
            questionCore: _questionCoreController.text.trim(),
            standardSolution: _standardSolutionController.text.trim(),
            wrongSolution: _wrongSolutionController.text.trim(),
          ),
          flowMode: draft.flowMode,
          sourceType: draft.sourceType,
        );
  }

  Future<void> _analyze() async {
    if (_questionCoreController.text.trim().isEmpty) {
      _showMessage('请先确认题目主干');
      return;
    }

    final draft = ref.read(questionDraftControllerProvider);
    if (draft == null ||
        draft.providerName.trim().isEmpty ||
        draft.modelName.trim().isEmpty) {
      _showMessage('请先选择 AI 模型厂商和模型名称');
      return;
    }

    _persistDraft();

    setState(() {
      _submitting = true;
    });

    try {
      await ref.read(questionFlowServiceProvider).analyzeCurrentDraft();
      if (mounted) {
        context.push('/questions/ai-review');
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

  void _showMessage(String message) {
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }

  void _confirmDiscard() {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('丢弃草稿'),
        content: const Text('确定要丢弃当前草稿吗？丢弃后无法恢复。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('取消'),
          ),
          TextButton(
            onPressed: () {
              Navigator.of(ctx).pop();
              ref.read(questionDraftControllerProvider.notifier).clear();
              context.go('/dashboard');
            },
            child: const Text('确定丢弃'),
          ),
        ],
      ),
    );
  }
}
