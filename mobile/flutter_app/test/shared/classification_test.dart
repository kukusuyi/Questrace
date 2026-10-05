import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:questrace_flutter/shared/models/classification.dart';
import 'package:questrace_flutter/shared/models/question_models.dart';
import 'package:questrace_flutter/shared/widgets/subject_picker.dart';

void main() {
  test(
    'classification survives draft serialization and payload projection',
    () {
      const c = Classification(
        subjectId: 'cs408',
        courseId: 'operating_systems',
        status: 'confirmed',
        subjectName: '408',
      );
      const draft = QuestionDraft(
        classification: c,
        subject: '408',
        chapter: '内存管理',
      );
      final restored = QuestionDraft.fromJson(draft.toJson());
      expect(restored.classification.subjectId, 'cs408');
      expect(restored.classification.courseId, 'operating_systems');
      expect(restored.toCreatePayload().toJson()['subject_id'], 'cs408');
      expect(QuestionDraft.emptyManual().classification.subjectId, '');
    },
  );
  test('filters keep scope in equality and query parameters', () {
    const a = ListQuestionFilter(
      classification: Classification(
        subjectId: 'cs408',
        courseId: 'data_structures',
        status: '',
      ),
    );
    final b = a.copyWith(
      classification: const Classification(
        subjectId: 'cs408',
        courseId: 'operating_systems',
        status: '',
      ),
    );
    expect(a == b, false);
    expect(b.toQueryParameters()['course_id'], 'operating_systems');
    expect(
      const ListQuestionFilter().toQueryParameters().containsKey(
        'classification_status',
      ),
      false,
    );
  });
  for (final size in [const Size(375, 812), const Size(812, 375)]) {
    for (final brightness in Brightness.values) {
      testWidgets('picker fits $size in $brightness with large text', (
        tester,
      ) async {
        tester.view.physicalSize = size;
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);
        var value = const Classification(
          subjectId: 'cs408',
          courseId: 'data_structures',
          subjectName: '408',
        );
        String chapter = '排序';
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              subjectCatalogProvider.overrideWith(
                (ref) async => const [
                  SubjectItem(
                    id: 'cs408',
                    name: '408',
                    specialized: true,
                    courses: [
                      SubjectCourse(
                        id: 'data_structures',
                        name: '数据结构',
                        chapters: ['排序'],
                      ),
                      SubjectCourse(
                        id: 'operating_systems',
                        name: '操作系统',
                        chapters: ['内存管理'],
                      ),
                    ],
                    chapters: [],
                  ),
                ],
              ),
            ],
            child: MaterialApp(
              theme: ThemeData(brightness: brightness),
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(context).copyWith(
                  textScaler: const TextScaler.linear(1.5),
                  disableAnimations: true,
                ),
                child: child!,
              ),
              home: Scaffold(
                body: SafeArea(
                  child: SingleChildScrollView(
                    child: Padding(
                      padding: const EdgeInsets.all(20),
                      child: StatefulBuilder(
                        builder: (context, setState) => SubjectPicker(
                          value: value,
                          chapter: chapter,
                          onChanged: (v, c) => setState(() {
                            value = v;
                            chapter = c;
                          }),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(find.text('数据结构'), findsWidgets);
        expect(tester.takeException(), isNull);
      });
    }
  }
}
