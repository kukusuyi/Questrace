import '../../shared/models/classification.dart';
import '../../shared/widgets/subject_picker.dart';
import '../../shared/widgets/solution_ocr_button.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/network/api_exception.dart';
import '../../shared/models/common_models.dart';
import '../../shared/models/question_models.dart';
import '../../shared/utils/draft_navigation.dart';
import '../../shared/widgets/analysis_picker.dart';
import '../../shared/widgets/draft_page_scope.dart';
import '../../shared/widgets/question_json_editor.dart';
import '../question_review/question_flow_service.dart';
import 'question_draft_controller.dart';

class QuestionCreatePage extends ConsumerStatefulWidget {
  const QuestionCreatePage({super.key});

  @override
  ConsumerState<QuestionCreatePage> createState() => _QuestionCreatePageState();
}

class _QuestionCreatePageState extends ConsumerState<QuestionCreatePage> {
  final _subjectController = TextEditingController();
  final _chapterController = TextEditingController();
  final _questionCoreController = TextEditingController();
  final _standardSolutionController = TextEditingController();
  final _wrongSolutionController = TextEditingController();
  final _jsonController = TextEditingController();
  bool _hydrated = false;
  bool _submitting = false;
  bool _advanced = false;
  String? _error;
  String _operation = '';
  String? _jsonError;
  QuestionCreateMode _mode = QuestionCreateMode.form;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final draft = ref.read(questionDraftControllerProvider);
      if (!mounted || draft == null || draft.status == DraftStatus.saved) {
        return;
      }
      if (draft.flowMode != DraftFlowMode.manual) {
        context.go(routeForDraft(draft));
      }
    });
  }

  @override
  void dispose() {
    _subjectController.dispose();
    _chapterController.dispose();
    _questionCoreController.dispose();
    _standardSolutionController.dispose();
    _wrongSolutionController.dispose();
    _jsonController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final draft = ref.watch(questionDraftControllerProvider);
    _hydrateIfNeeded(draft);
    return DraftPageScope(
      child: Scaffold(
        appBar: AppBar(
          title: const Text('整理错题'),
          leading: BackButton(
            onPressed: () => Navigator.of(context).maybePop(),
          ),
          actions: [
            PopupMenuButton<String>(
              tooltip: '更多操作',
              onSelected: (v) {
                if (v == 'advanced') {
                  setState(() => _advanced = !_advanced);
                } else if (v == 'image') {
                  context.push('/questions/upload');
                } else {
                  _confirmDiscard();
                }
              },
              itemBuilder: (_) => const [
                PopupMenuItem(value: 'image', child: Text('改用图片录入')),
                PopupMenuItem(value: 'advanced', child: Text('高级编辑')),
                PopupMenuItem(value: 'discard', child: Text('清空草稿')),
              ],
            ),
          ],
        ),
        bottomNavigationBar: SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: _submitting
                        ? null
                        : () async {
                            _persistDraft();
                            if (await showAnalysisPicker(context) && mounted) {
                              await _analyze();
                            }
                          },
                    child: Text(
                      _submitting && _operation == 'analysis'
                          ? '分析中…'
                          : 'AI 辅助',
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    onPressed: _submitting ? null : _saveDirectly,
                    child: Text(
                      _submitting && _operation == 'save' ? '保存中…' : '保存错题',
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
        body: ListView(
          padding: const EdgeInsets.all(20),
          keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
          children: [
            if (_error != null)
              Padding(
                padding: const EdgeInsets.only(bottom: 16),
                child: Text(
                  _error!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
            if (_advanced) ...[
              SegmentedButton<QuestionCreateMode>(
                segments: const [
                  ButtonSegment(
                    value: QuestionCreateMode.form,
                    label: Text('表单'),
                  ),
                  ButtonSegment(
                    value: QuestionCreateMode.json,
                    label: Text('JSON'),
                  ),
                ],
                selected: {_mode},
                onSelectionChanged: (selection) {
                  final next = selection.first;
                  if (next == QuestionCreateMode.json) {
                    _syncJsonFromForm();
                  } else if (!_applyJsonToForm(showError: true)) {
                    return;
                  }
                  setState(() => _mode = next);
                },
              ),
              const SizedBox(height: 16),
            ],
            Card(
              child: Padding(
                padding: const EdgeInsets.all(20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '题目与思路',
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const SizedBox(height: 20),
                    if (_mode == QuestionCreateMode.form) ...[
                      TextFormField(
                        controller: _questionCoreController,
                        minLines: 4,
                        maxLines: 10,
                        decoration: const InputDecoration(
                          labelText: '题目内容',
                          hintText: '支持文字与 LaTeX 公式',
                        ),
                        onChanged: (_) => _persistDraft(),
                      ),
                      const SizedBox(height: 16),
                      TextFormField(
                        controller: _standardSolutionController,
                        minLines: 3,
                        maxLines: 8,
                        decoration: const InputDecoration(labelText: '标准解（可选）'),
                        onChanged: (_) => _persistDraft(),
                      ),
                      SolutionOcrButton(
                        controller: _standardSolutionController,
                        onApplied: _persistDraft,
                      ),
                      const SizedBox(height: 16),
                      TextFormField(
                        controller: _wrongSolutionController,
                        minLines: 3,
                        maxLines: 8,
                        decoration: const InputDecoration(
                          labelText: '错误思路（可选）',
                        ),
                        onChanged: (_) => _persistDraft(),
                      ),
                    ] else
                      QuestionJsonEditor(
                        controller: _jsonController,
                        errorText: _jsonError,
                        onChanged: (_) => _persistDraft(),
                      ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            Card(
              child: ExpansionTile(
                shape: const Border(),
                title: const Text('学科与章节'),
                subtitle: Text(
                  _subjectController.text.isEmpty
                      ? '可以稍后完善'
                      : _subjectController.text,
                ),
                childrenPadding: const EdgeInsets.all(20),
                children: [
                  SubjectPicker(
                    suggestedSubject: draft?.suggestedSubject ?? '',
                    onSuggestionDismissed: () => ref
                        .read(questionDraftControllerProvider.notifier)
                        .dismissSuggestedSubject(),
                    value: draft?.classification ?? const Classification(),
                    chapter: _chapterController.text,
                    onChanged: (v, c) {
                      _subjectController.text = v.subjectName;
                      _chapterController.text = c;
                      ref
                          .read(questionDraftControllerProvider.notifier)
                          .updateClassification(v, c);
                    },
                  ),
                ],
              ),
            ),
            if (draft?.sourceImageUrl.isNotEmpty == true)
              const Padding(
                padding: EdgeInsets.all(16),
                child: Text('已附上原图，保存后可在详情中查看。'),
              ),
          ],
        ),
      ),
    );
  }

  void _hydrateIfNeeded(QuestionDraft? draft) {
    if (_hydrated || draft == null) {
      return;
    }

    _subjectController.text = draft.subject;
    _chapterController.text = draft.chapter;
    _questionCoreController.text = draft.questionJson.questionCore;
    _standardSolutionController.text = draft.questionJson.standardSolution;
    _wrongSolutionController.text = draft.questionJson.wrongSolution;
    _jsonController.text = formatQuestionJson(draft.questionJson);
    _hydrated = true;
  }

  void _persistDraft() {
    final questionJson = _currentQuestionJson();
    if (questionJson == null) {
      return;
    }

    ref
        .read(questionDraftControllerProvider.notifier)
        .updateBasicFields(
          subject: _subjectController.text.trim(),
          chapter: _chapterController.text.trim(),
          questionJson: questionJson,
          flowMode: DraftFlowMode.manual,
          sourceType:
              ref.read(questionDraftControllerProvider)?.sourceImageId != null
              ? SourceType.image
              : SourceType.manual,
        );
  }

  Future<void> _saveDirectly() async {
    if (_submitting) return;
    _operation = 'save';
    _error = null;
    if (_mode == QuestionCreateMode.json &&
        !_applyJsonToForm(showError: true)) {
      return;
    }
    if (_questionCoreController.text.trim().isEmpty) {
      _showMessage('请先填写题目主干');
      return;
    }
    _persistDraft();
    setState(() {
      _submitting = true;
    });
    try {
      final id = await ref.read(questionFlowServiceProvider).saveCurrentDraft();
      if (mounted) context.go('/questions/$id');
    } catch (e) {
      if (mounted) setState(() => _error = describeError(e));
    } finally {
      if (mounted) {
        setState(() {
          _submitting = false;
        });
      }
    }
  }

  Future<void> _analyze() async {
    if (_submitting) return;
    _operation = 'analysis';
    _error = null;
    if (_mode == QuestionCreateMode.json &&
        !_applyJsonToForm(showError: true)) {
      return;
    }

    final draft = ref.read(questionDraftControllerProvider);
    if (draft == null ||
        draft.providerName.trim().isEmpty ||
        draft.modelName.trim().isEmpty) {
      _showMessage('请先选择 AI 模型厂商和模型名称');
      return;
    }

    if (_questionCoreController.text.trim().isEmpty) {
      _showMessage('请先填写题目主干');
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

  QuestionJson? _currentQuestionJson() {
    if (_mode == QuestionCreateMode.json) {
      try {
        final parsed =
            tryParseQuestionJson(_jsonController.text) ?? const QuestionJson();
        if (_jsonError != null) {
          setState(() {
            _jsonError = null;
          });
        }
        return parsed;
      } on FormatException catch (error) {
        if (_jsonError != error.message) {
          setState(() {
            _jsonError = error.message;
          });
        }
        return null;
      } on Object {
        const message = 'JSON 格式错误，请检查花括号、引号和字段名';
        if (_jsonError != message) {
          setState(() {
            _jsonError = message;
          });
        }
        return null;
      }
    }

    if (_jsonError != null) {
      setState(() {
        _jsonError = null;
      });
    }
    return QuestionJson(
      questionCore: _questionCoreController.text.trim(),
      standardSolution: _standardSolutionController.text.trim(),
      wrongSolution: _wrongSolutionController.text.trim(),
    );
  }

  bool _applyJsonToForm({required bool showError}) {
    try {
      final parsed =
          tryParseQuestionJson(_jsonController.text) ?? const QuestionJson();
      _questionCoreController.text = parsed.questionCore;
      _standardSolutionController.text = parsed.standardSolution;
      _wrongSolutionController.text = parsed.wrongSolution;
      if (_jsonError != null) {
        setState(() {
          _jsonError = null;
        });
      }
      _persistDraft();
      return true;
    } on FormatException catch (error) {
      if (showError) {
        setState(() {
          _jsonError = error.message;
        });
        _showMessage(error.message);
      }
      return false;
    } on Object {
      const message = 'JSON 格式错误，请检查花括号、引号和字段名';
      if (showError) {
        setState(() {
          _jsonError = message;
        });
        _showMessage(message);
      }
      return false;
    }
  }

  void _syncJsonFromForm() {
    final raw = formatQuestionJson(
      QuestionJson(
        questionCore: _questionCoreController.text.trim(),
        standardSolution: _standardSolutionController.text.trim(),
        wrongSolution: _wrongSolutionController.text.trim(),
      ),
    );

    if (_jsonController.text != raw) {
      _jsonController.text = raw;
    }
    if (_jsonError != null) {
      setState(() {
        _jsonError = null;
      });
    }
  }
}

enum QuestionCreateMode { form, json }
