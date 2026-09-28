import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../shared/models/ai_models.dart';
import '../../shared/models/common_models.dart';
import '../../shared/models/file_models.dart';
import '../question_create/question_draft_controller.dart';
import '../question_list/question_repository.dart';
import 'ai_repository.dart';

final questionFlowServiceProvider = Provider<QuestionFlowService>((ref) {
  return QuestionFlowService(ref);
});

class QuestionFlowService {
  QuestionFlowService(this._ref);

  final Ref _ref;

  Future<void> recognizeUploadedImage(UploadedImage image) async {
    final draftController = _ref.read(questionDraftControllerProvider.notifier);
    draftController.replaceWithUpload(image);
    await draftController.flush();
    draftController.markStatus(DraftStatus.ocrProcessing);
    await draftController.flush();

    final result = await _ref
        .read(aiRepositoryProvider)
        .recognizeWrongQuestion(
          imageUrl: image.imageUrl,
          imageId: image.imageId,
        );

    draftController.setOcrResult(
      questionJson: result.questionJson,
      ocrContext: result.ocrContext,
    );
  }

  Future<void> analyzeCurrentDraft() async {
    final draftController = _ref.read(questionDraftControllerProvider.notifier);
    await draftController.flush();
    final draft = _ref.read(questionDraftControllerProvider);
    if (draft == null) {
      throw StateError('当前没有可分析的草稿');
    }

    draftController.markStatus(DraftStatus.aiProcessing);
    await draftController.flush();

    final result = await _ref
        .read(aiRepositoryProvider)
        .analyzeWrongQuestion(
          AnalyzeWrongQuestionPayload(
            classification: draft.classification,
            providerName: draft.providerName.isEmpty
                ? null
                : draft.providerName,
            modelName: draft.modelName.isEmpty ? null : draft.modelName,
            chapter: draft.chapterLocked && draft.chapter.isNotEmpty
                ? draft.chapter
                : null,
            questionJson: draft.questionJson,
            ocrContext: draft.ocrContext,
          ),
        );

    draftController.applyAiAnalysis(result);
  }

  Future<int> saveCurrentDraft() async {
    final draftController = _ref.read(questionDraftControllerProvider.notifier);
    await draftController.flush();
    final draft = _ref.read(questionDraftControllerProvider);
    if (draft == null) {
      throw StateError('当前没有可保存的草稿');
    }

    final result = await _ref
        .read(questionRepositoryProvider)
        .createQuestion(draft.toCreatePayload());

    draftController.markStatus(DraftStatus.saved);
    draftController.clear();

    return result.questionId;
  }
}
