import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:questrace_flutter/core/network/api_client.dart';
import 'package:questrace_flutter/features/profile/profile_card.dart';
import 'package:questrace_flutter/shared/models/auth_models.dart';
import 'package:questrace_flutter/shared/models/classification.dart';
import 'package:questrace_flutter/shared/widgets/subject_picker.dart';

void main() {
  test('registration carries the selected education stage', () {
    const payload = RegisterPayload(
      username: 'student',
      password: 'test-password',
      email: 'test@example.invalid',
      educationStage: 'highschool',
    );
    expect(payload.toJson()['education_stage'], 'highschool');
  });
  testWidgets('personal information saves stage and reloads the profile', (
    tester,
  ) async {
    var stage = 'university';
    var saved = false;
    final dio = Dio()
      ..interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            expect(options.path, '/api/v1/users/me');
            expect(options.method, 'PUT');
            stage = options.data['education_stage'] as String;
            saved = true;
            handler.resolve(
              Response(
                requestOptions: options,
                data: {'education_stage': stage},
                statusCode: 200,
              ),
            );
          },
        ),
      );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          apiClientProvider.overrideWithValue(dio),
          userProfileProvider.overrideWith(
            (ref) async => {
              'username': 'student',
              'email': 'test@example.invalid',
              'education_stage': stage,
            },
          ),
        ],
        child: const MaterialApp(
          home: Scaffold(body: SingleChildScrollView(child: ProfileCard())),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('大学'), findsOneWidget);
    await tester.tap(find.byType(DropdownButtonFormField<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('高中').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('保存学习阶段'));
    await tester.pumpAndSettle();
    expect(saved, isTrue);
    expect(stage, 'highschool');
    expect(find.text('高中'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
  testWidgets('highschool recommendations hide university-only branches', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          subjectCatalogProvider.overrideWith(
            (ref) async => const [
              SubjectItem(
                id: 'cs408',
                name: '408',
                specialized: true,
                courses: [],
                chapters: [],
                recommended: false,
              ),
              SubjectItem(
                id: 'highschool_math',
                name: '高中数学',
                specialized: false,
                courses: [],
                chapters: [],
                educationStage: 'highschool',
              ),
            ],
          ),
        ],
        child: MaterialApp(
          home: Scaffold(
            body: SubjectPicker(
              value: const Classification(),
              chapter: '',
              onChanged: (c, s) {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownMenu<String>));
    await tester.pumpAndSettle();
    expect(find.text('高中数学'), findsWidgets);
    expect(find.text('408'), findsNothing);
    expect(tester.takeException(), isNull);
  });
}
