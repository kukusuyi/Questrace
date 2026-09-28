import 'package:questrace_flutter/shared/models/classification.dart';
import 'dart:convert';
import 'package:questrace_flutter/shared/models/ai_models.dart';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:questrace_flutter/core/storage/key_value_store.dart';
import 'package:questrace_flutter/core/storage/storage_keys.dart';
import 'package:questrace_flutter/features/question_create/question_draft_controller.dart';
import 'package:questrace_flutter/shared/models/common_models.dart';
import 'package:questrace_flutter/shared/models/question_models.dart';
import 'package:questrace_flutter/shared/utils/draft_navigation.dart';

void main() {
  test(
    'rejecting repeated AI analysis restores original draft after restart',
    () async {
      SharedPreferences.setMockInitialValues({});
      final prefs = await SharedPreferences.getInstance();
      ProviderContainer make() => ProviderContainer(
        overrides: [sharedPreferencesProvider.overrideWithValue(prefs)],
      );
      final first = make();
      final controller = first.read(questionDraftControllerProvider.notifier);
      controller.updateBasicFields(
        subject: '高中物理',
        chapter: '原章节',
        questionJson: const QuestionJson(questionCore: '原题目'),
      );
      controller.updateClassification(
        const Classification(
          subjectId: 'highschool_physics',
          subjectName: '高中物理',
          status: 'confirmed',
        ),
        '原章节',
      );
      const result = AnalyzeWrongQuestionResponse(
        classification: Classification(
          subjectId: 'cs408',
          courseId: 'operating_systems',
          subjectName: '408',
          status: 'confirmed',
        ),
        chapter: 'AI章节',
        tags: TagGroups(),
        semanticSummary: 'AI摘要',
        mistakeSummary: 'AI错因',
      );
      controller.applyAiAnalysis(result);
      controller.applyAiAnalysis(result);
      await controller.flush();
      first.dispose();
      final second = make();
      addTearDown(second.dispose);
      await second
          .read(questionDraftControllerProvider.notifier)
          .discardAnalysis();
      expect(
        second.read(questionDraftControllerProvider)!.classification.subjectId,
        'highschool_physics',
      );
      expect(
        second.read(questionDraftControllerProvider)!.classification.courseId,
        '',
      );
      expect(second.read(questionDraftControllerProvider)!.chapter, '原章节');
      expect(
        second.read(questionDraftControllerProvider)!.questionJson.questionCore,
        '原题目',
      );
      expect(
        second.read(questionDraftControllerProvider)!.status,
        DraftStatus.draft,
      );
    },
  );
  test(
    'recovers processing drafts back to OCR review and persists them',
    () async {
      const processingDraft = QuestionDraft(
        flowMode: DraftFlowMode.upload,
        sourceType: SourceType.image,
        sourceImageId: 7,
        sourceImageUrl: 'https://example.com/question.png',
        questionJson: QuestionJson(questionCore: r'x^2+1=0'),
        status: DraftStatus.aiProcessing,
      );

      SharedPreferences.setMockInitialValues({
        '${StorageKeys.questionDraft}:': jsonEncode(processingDraft.toJson()),
      });
      final preferences = await SharedPreferences.getInstance();
      final container = ProviderContainer(
        overrides: [sharedPreferencesProvider.overrideWithValue(preferences)],
      );
      addTearDown(container.dispose);

      final recoveredDraft = container.read(questionDraftControllerProvider);

      expect(recoveredDraft, isNotNull);
      expect(recoveredDraft!.status, DraftStatus.ocrReviewing);
      expect(routeForDraft(recoveredDraft), '/questions/ocr-review');

      await container.read(questionDraftControllerProvider.notifier).flush();

      final persistedDraft = QuestionDraft.fromJson(
        jsonDecode(preferences.getString('${StorageKeys.questionDraft}:')!)
            as Map<String, dynamic>,
      );
      expect(persistedDraft.status, DraftStatus.ocrReviewing);
    },
  );
}
